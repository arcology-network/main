<!-- # main  -->
<h1> main <img align="center" height="50" src="./img/three-square.svg">  </h1>

The main project contains all the service modules for the Arcology network. These modules are encapsulated as Actors within the Streamer framework, an inter-EDA-based development framework. These modules can be deployed as either independent threads or services on different machines.


<h2> Modules <img align="center" height="32" src="./img/cloud.svg">  </h2>

 Name           | Description                                                                                                                |
|------------------|----------------------------------------------------------------------------------------------------------------------------|
| Arbitrator       | Functions as a wrapper to identify state access conflicts among transitions for conflict detection.                       |
| Consensus Module | Manages logic related to consensus within the Arcology system.                                                              |
| Core             | Governs the block cycle and coordinates consensus with other modules.                                                      |
| Eth-API          | Provides support for the Eth RPC API.                                                                                      |
| Exec             | Encompasses multiple Execution Units (EUs) and communication modules.                                                      |
| Pool             | Oversees the transaction pool for an Arcology node.                                                                        |
| Scheduler        | Orchestrates the scheduling of transaction input branch batches and generations for optimal execution efficiency. Also maintains a conflict history for optimized schedules. |
| Sync             | Houses inter-node synchronization modules for synchronizing blocks, receipts, states, and conflict histories among Arcology nodes. |
| Gateway          | Manages transitions originating from different sources.                                                                    |
| TPP              | Primarily validates the signature of transitions received by the gateway.                                                  |                                                                                                                |

<h2> Getting Started <img align="center" height="32" src="./img/running2.svg">  </h2>

- **Binary release :** Please download the latest [binary releases](https://github.com/arcology-network/binary-releases/tags). 

- **Build From Source:** 


<h2> Configuration <img align="center" height="32" src="./img/ruler-cross-pen.svg">  </h2>

Arcology offers two major Installation options with distinct focuses. The modules can with be deployed as independent threads or services on different machines. The intra-process option emphasizes optimizing resource utilization for efficiency, while the inter-process option is geared towards achieving superior performance.

 <h3> 1. Intra-process(Inter-thread) <img align="center" height="25" src="./img/share-circle.svg">  </h3>

The intra-process is the most cost-efficient deployment choice. In this mode, all the major modules are deployed as threads connected by an event broker. Since all the modules reside in the same process, this type of deployment effectively avoids the overhead associated with inter-process communication.

 <h3>2. Inter-processs(Cluster Deployment)   <img align="center" height="25" src="./img/msg.svg">  </h3>

In this mode, modules are deployed across multiple interconnected machines to achieve maximum performance. It is also referred to as cluster deployment. Arcology comes with tools to assist in this process. An Arcology cluster can either be deployed on a local machine, which refers to On-premise mode or on a Amazon AWS 

1. [On-premise]() 

2. [On AWS](https://github.com/arcology-network/aws-ansible)

>> These scripts are developed and tested on Ubuntu 22.04 LTS only 


<h2> License  <img align="center" height="32" src="./img/copyright.svg">  </h2>

This project is licensed under the MIT License.
