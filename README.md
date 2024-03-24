# KCore

## Overview

KCore is a highly efficient implementation of the core features of the Apache Kafka protocol. Designed and developed in Go, KCore offers an ultra-low-footprint, high-performance event streaming platform. It is tailor-made for applications requiring real-time data streaming in environments where resources are constrained and performance is of the essence.

### Key Features

> [!WARNING]
> KCore is currently under development and not all the following features are implemented. The project is not yet ready
> for production use. Please check back for updates on our progress.

- **Low Resource Consumption**: Optimized for environments with limited resources without compromising on performance.
- **High Performance**: Engineered to provide fast, reliable streaming capabilities.
- **Apache Kafka Core Features**: Supports essential features such as the producer and consumer API, topic replication, and management.
- **Scalability and Fault-Tolerance**: Inherits Kafka's renowned scalability and fault-tolerance to ensure reliable event handling.
- **Versatile Application**: Ideal for edge computing, IoT, automotive, and more.

## Use Cases

KCore is the perfect solution for projects that demand the scalability and fault-tolerance of Apache Kafka but operate in environments unsuitable for a full Kafka cluster. Its applications include, but are not limited to:

- Edge Computing
- Internet of Things (IoT)
- Automotive Systems
- Any application requiring real-time data streaming without the overhead of a full Apache Kafka cluster.

## Why KCore?

The motivation behind KCore stems from the need for a low-footprint, efficient alternative to Apache Kafka. Many applications do not require the extensive feature set of Kafka, such as the Kafka Streams API. However, they still need a reliable and scalable event streaming platform. KCore fills this gap by offering the most critical capabilities of Apache Kafka with significantly reduced complexity and resource consumption.

## Roadmap

KCore is currently under development. The following features are planned for future releases:
- [ ] Producer API
- [ ] Consumer API
- [ ] Topic Replication
- [ ] Topic Management
- [ ] Encryption and Authentication

## Getting Started

To get started with KCore, clone the repository and follow the setup instructions.

### Prerequisites

- Go (version 1.21 or later)
- [Mage](https://magefile.org/) - Mage is a Make/rake-like build tool using Go. 
- 

### Installing

Clone the repository:

```bash
git clone git@github.com:kcore-io/kcore.git
mage build
./kcore
```

## Documentation

For more detailed information about KCore's capabilities and how to use them, please refer to the documentation.

## Contributing
We welcome contributions from the community! If you'd like to contribute to KCore, please see our contributing guidelines for more information.

## Trademark Notice

[Apache Kafka](https://kafka.apache.org/) is a registered trademark of the [Apache Software Foundation](https://www.apache.org/). KCore is not affiliated with, endorsed by, or sponsored by the Apache Software Foundation. We are an independent project that aims to provide a lightweight complementary solution in the Apache Kafka ecosystem.

## License
KCores is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.
