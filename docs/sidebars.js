/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  // By default, Docusaurus generates a sidebar from the docs folder structure
  packetstreamer: [
    {
      type: 'html',
      value: 'Deepfence PacketStreamer',
      className: 'sidebar-title',
    },

    'packetstreamer/index',
    'packetstreamer/quickstart',

    {
      type: 'category',
      label: 'Using PacketStreamer',

      items: [
        'packetstreamer/using/build',
        'packetstreamer/using/local',
        'packetstreamer/using/docker',
        'packetstreamer/using/kubernetes',
        'packetstreamer/using/vagrant',
      ]
    },

    'packetstreamer/configuration',

    {
      type: 'category',
      label: 'Integrations',
      link: {
        type: 'generated-index',
        description:
          "Integrations and Plugins can extend PacketStreamer"
      },
      items: [
        'packetstreamer/extra/s3',
        'packetstreamer/extra/suricata',
      ]
    }
  ]
};

module.exports = sidebars;
