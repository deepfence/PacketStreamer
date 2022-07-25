# Plugins

This documentation section is about plugins which allow to stream packets to
various external storage services.

Plugins can be used both from:

- **sensor** - in that case, locally captured packets are streamed through the
  plugin
- **receiver** - all packets retrieved from (potentially multiple) sensors are
  streamed through the plugin

Currently the plugins are:

- [S3](./s3.md)
