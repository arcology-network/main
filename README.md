# main 

The main project contains all the service modules for the Arcology network. These modules are encapsulated as Actors within the Streamer framework, an inter-EDA-based development framework. These modules can be deployed as either independent threads or services on different machines.

## Getting Started

- **Binary release :** Please download the latest [binary releases](https://github.com/arcology-network/binary-releases/tags). 

- **Build From Source:**

## Installation

Arcology offers two major Installation options with distinct focuses. The modules can be deployed as independent threads or services on different machines. The intra-process option emphasizes optimizing resource utilization for efficiency, while the inter-process option is geared towards achieving superior performance.

### 1. Intra-process(Inter-thread)

The intra-process is the most cost-efficient deployment choice. In this mode, all the major modules are deployed as threads connected by an event broker. Since all the modules reside in the same process, this type of deployment effectively avoids the overhead associated with inter-process communication.

### 2. Inter-processs(Cluster Deployment)

In this mode, modules are deployed across multiple interconnected machines to achieve maximum performance. It is also referred to as cluster deployment. Arcology comes with tools to assist in this process. An Arcology cluster can either be deployed on a local machine, which refers to On-premise mode or on a Amazon AWS 

1. [On-premise]() 

2. [On AWS](https://github.com/arcology-network/aws-ansible)

>> These scripts are developed and tested on Ubuntu 22.04 LTS only 

## Usage

Instructions on how to use the project or any relevant examples.

## License

This project is licensed under the MIT License.
