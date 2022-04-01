package streamer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"database/sql"

	"github.com/google/gopacket"
)

const authBuffSize = 64
const keyLenSize = 2
const respLen = 5
const respIdx = 4

type authConnIntf interface {
	SetReadDeadline(t time.Time) error
	Write(b []byte) (int, error)
	Read(b []byte) (int, error)
}

type dfPkt interface {
	ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error)
}

func handleClientAuth(conn authConnIntf, authKey string) error {

	var authData [authBuffSize]byte
	var keyLen = make([]byte, keyLenSize)
	var bytesWritten = 0
	var bytesToWrite = 0
	var bytesRead = 0
	var authBytes = []byte(authKey)
	var respBuff [respLen]byte

	binary.LittleEndian.PutUint16(keyLen, uint16(len(authBytes)))
	copy(authData[bytesWritten:], hdrData[:])
	bytesWritten += len(hdrData)
	copy(authData[bytesWritten:], keyLen[:])
	bytesWritten += keyLenSize
	copy(authData[bytesWritten:], authBytes[:])
	for {
		bytesWritten, err := conn.Write(authData[bytesToWrite:])
		if err != nil {
			errStr := fmt.Sprintf("Unable to send auth data to server. Reason %s \n", err.Error())
			return errors.New(errStr)
		}
		if bytesWritten == 0 {
			return errors.New("No bytes sent to server. Bailing out ")
		}
		bytesToWrite += bytesWritten
		if bytesToWrite == authBuffSize {
			break
		}
	}
	for {
		deadLineErr := conn.SetReadDeadline(time.Now().Add(connTimeout * time.Second))
		if deadLineErr != nil {
			log.Println("Unable to set read deadline ", deadLineErr.Error())
		}
		currBytesRead, readErr := conn.Read(respBuff[bytesRead:])
		if (readErr != nil) && (readErr != io.EOF) && (os.IsTimeout(readErr) == false) {
			return errors.New("Server closed connection " + readErr.Error())
		}
		bytesRead += currBytesRead
		if bytesRead == respLen {
			break
		}
		if (readErr == io.EOF) && (bytesRead != respLen) {
			return errors.New("Server closed connection abruptly. Got EOF")
		}
		if (os.IsTimeout(readErr) == true) && (bytesRead != respLen) {
			return errors.New("Server timed out.")
		}
	}
	compareRes := bytes.Compare(respBuff[0:len(hdrData)], hdrData[:])
	if compareRes != 0 {
		return errors.New("Illegal response received from server")
	}
	if respBuff[respIdx] != 0x0 {
		return errors.New("Authenticated declined by server ")
	}
	return nil
}

func handleServerAuth(hostConn net.Conn) bool {

	var userId string
	var authSuccess bool
	var authData [authBuffSize]byte
	var respData [respLen]byte
	totalBytesRead := 0
	totalBytesWritten := 0

	for {
		deadLineErr := hostConn.SetReadDeadline(time.Now().Add(connTimeout * time.Second))
		if deadLineErr != nil {
			log.Println(fmt.Sprintf("Unable to set timeout for connection from %s. Reason %s",
				hostConn.RemoteAddr(), deadLineErr.Error()))
		}
		bytesRead, readErr := hostConn.Read(authData[totalBytesRead:])
		if (readErr != nil) && (readErr != io.EOF) && (os.IsTimeout(readErr) == false) {
			log.Println(fmt.Sprintf("Client %s closed connection. Reason = %s ",
				hostConn.RemoteAddr(), readErr.Error()))
			hostConn.Close()
			return false
		}
		totalBytesRead += bytesRead
		if totalBytesRead == authBuffSize {
			break
		}
		if (readErr == io.EOF) && (totalBytesRead != authBuffSize) {
			log.Println(fmt.Sprintf("Client %s closed connection abruptly. Got EOF", hostConn.RemoteAddr()))
			hostConn.Close()
			return false
		}
		if (os.IsTimeout(readErr) == true) && (totalBytesRead != authBuffSize) {
			log.Println(fmt.Sprintf("Client %s timed out", hostConn.RemoteAddr()))
			hostConn.Close()
			return false
		}
	}
	totalBytesRead = 0
	compareRes := bytes.Compare(authData[totalBytesRead:len(hdrData)], hdrData[:])
	if compareRes != 0 {
		log.Println("Unknown header received. Disconnecting ", hostConn.RemoteAddr())
		hostConn.Close()
		return false
	}

	totalBytesRead += len(hdrData)
	keyLen := binary.LittleEndian.Uint16(authData[totalBytesRead:])
	totalBytesRead += keyLenSize
	userId, authSuccess =
		checkAuth(string(authData[totalBytesRead : totalBytesRead+int(keyLen)]))
	if authSuccess == false {
		log.Println("Error while authenticating client", userId)
		hostConn.Close()
		return false
	}
	copy(respData[0:], hdrData[:])
	respData[respLen-1] = 0x0
	for {
		bytesWritten, writeErr := hostConn.Write(respData[totalBytesWritten:])
		if writeErr != nil {
			log.Println("Unable to send data to client ", hostConn.RemoteAddr())
			hostConn.Close()
			return false
		}
		totalBytesWritten += bytesWritten
		if totalBytesWritten == respLen {
			break
		}
	}

	return true
}

func checkAuth(authString string) (string, bool) {

	var email string
	pgHost := os.Getenv("POSTGRES_USER_DB_HOST")
	pgPort := os.Getenv("POSTGRES_USER_DB_PORT")
	pgUser := os.Getenv("POSTGRES_USER_DB_USER")
	pgPwd := os.Getenv("POSTGRES_USER_DB_PASSWORD")
	pgDbName := os.Getenv("POSTGRES_USER_DB_NAME")
	pgSslMode := os.Getenv("POSTGRES_USER_DB_SSLMODE")
	if (pgHost == "") || (pgPort == "") || (pgUser == "") ||
		(pgPwd == "") || (pgDbName == "") || (pgSslMode == "") {
		return "Missing credentials", false
	}
	portVal, portErr := strconv.Atoi(pgPort)
	if portErr != nil {
		return portErr.Error(), false
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pgHost, portVal, pgUser, pgPwd, pgDbName, pgSslMode)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err.Error(), false
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		return err.Error(), false
	}
	sqlQuery := `SELECT email FROM "user" WHERE api_key=$1;`
	rowVal := db.QueryRow(sqlQuery, authString)
	err = rowVal.Scan(&email)
	if err != nil {
		return err.Error(), false
	}
	return email, true
}
