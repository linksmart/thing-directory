---
title: 'Thing Directory: A lightweight registry of WoT Thing Descriptions'
tags:
authors:
  - name: Farshid Tavakolizadeh
    affiliation: 1
  - name: Shreekantha Devasya
    affiliation: 1
affiliations:
 - name: Fraunhofer Institute for Applied Information Technology, Sankt Augustin, Germany
   index: 1
date: 16 December 2020
bibliography: paper.bib

---

# Statement of Need  
<!-- A clear Statement of Need that illustrates the research purpose of the software. -->

The fast emergence of IoT (internet of things) technologies has influenced scientific communities to embrace novel sources of information and their potential use in various domains. While the vast amount of sensory data is beneficial, the lack of uniform access interfaces hinders researchers from efficient exploitation. A structured yet flexible registry is needed to model device metadata and allow exploration and interaction with such devices. In the IoT context, the devices are physical things (e.g., sensors, actuators, gateways) or virtual ones (e.g., digital twins, proxies, aggregated data sources). An ideal metadata registry for such devices should have a flat learning curve and be easily deployable. This would allow researchers to focus less on interoperability and more on fast prototyping and data application. Registries which are based on established standards are preferred since they can incorporate metadata of with many existing devices out-of-the-box. Moreover, the registry software be lightweight, easily deployable, and free of rigid requirements to support fast prototyping across a wide range of use cases from edge to cloud. 

# Related Work 
<!-- A list of key references, including to other software addressing related needs. -->

There are multiple attempts to modelling and discovering the connected Things and their interfaces. OGC Sensorthings API [@sensorthingsSensing, @sensorthingsTasking] has been a successful model for representation of Things. Sensorthings API consists of two parts: sensing and tasking. The popular implementations of the OGC SensorThings Models are FROST [@frost] and GOST [@gost] support mainly the sensing part. FROST has a preliminary tasking support. The solutions are focused on rather centralized deployments and are not suitable for a federated scenario and for the gateways with limited computational power. The specification enforces both metadata and observation modeling which is not practical in many IoT scenarios having different data formats and interfaces. The other technologies related to Discovery in IoT field are discussed and categorized by Bröring et. al [@DiscoveryCategorization2016]. Web of Things Thing Description (WoT TD) [@WoTTD] covers wide range of IoT devices by providing the semantics to describe the textual metadata, interaction affordances, data schemas and relations. It also supports semantic annotations by providing JSON-LD encoding. Thingweb Directory [@Thingweb] has implemented the discovery of WoT TDs described using a proprietary API. Unlike these implementations, the Thing Directory follows the [@WoTDiscovery] standard.  


# Thing Directory 
<!-- A summary describing the high-level functionality and purpose of the software for a diverse, non-specialist audience. -->

Thing Directory is an open-source implementation of the W3C WoT Thing Description Directory [@WoTDiscovery], a searchable directory of metadata for things. It is based on a modular architecture with pluggable storage backend, enabling selection of different database systems depending on the application runtime requirements. The current implementation, backed by an embedded LevelDB storage, can be deployed on highly constrained single-board computers such as the Raspberry Pi Zero series. It has very minimal idle processing and memory footprints and can scale on demand to utilize all locally available resources. More powerful storage backends can be added to create a horizontally scalable directory in cloud environments. 

The application is packaged as binary distributions, Debian packages, as well as Docker images for easy deployment on a variety of platforms. The data model of the server is based on W3C WoT Thing Description [@WoTTD] which is extensible by design. Thing Directory validates all inputs using a JSON-Schema, describing the data model. This makes it possible to extend server’s structured data model and input validation mechanism with zero programming. The various modules of the system are provided as re-usable Go libraries which can be imported by other applications to build functionalities around Thing Descriptions.  
 

<!-- Mention (if applicable) a representative set of past or ongoing research projects using the software and recent scholarly publications enabled by it. -->

## Use case: Assessment of Building Energy Efficiency 
Construction companies often deal with the challenge of delivering target energy-efficient buildings given specific budget and time constraints. Energy efficiency, as one of the key factors for renovation investments, depends on the availability of various data sources to support the renovation design and planning. These include climate data and building material along with residential comfort and energy consumption patterns. 

As part of the pre-renovation activities, the construction companies deploy various sensors to collect relevant data over a period. Such sensors become part of a wireless sensor network (WSN) and expose data endpoint with the help of one or more gateway devices. Depending on the protocols, the endpoints require different interaction flows to securely access the current and historical measurements. The renovation applications need to discover the sensors, their endpoints and how to interact with them based on search criteria such as the physical location, mapping to the building model or measurement type. 

The Thing Directory supports scientists in assessment of building energy efficiency by providing the means to collect and discover the metadata of deployed sensors in an easy and standardized way. Instances of Thing Directory have been deployed in four renovation sites (two apartments, two buildings) across Europe as registries of wireless sensors which are accessible over Z-Wave or WiFi. The API has been integrated into components for Profiling of Resident Usage of Building Systems (PRUBS), Process & Workflow Modelling & Automation (PWMA), and Building Information Collection Application (BICA). These applications query sensor metadata based on zoning and sensor types. Once discovered, the metadata provides these applications with necessary details on how to authenticate and query data. Since the Thing Directory is based on an open standard, being integrated with it adds interoperability with WSNs beyond the scope of this use case. The applications will be able to interact with the growing number of compliant WoT devices.  


# Acknowledgement 
<!-- Acknowledgement of any financial support. -->

This work was conducted as part of the BIMERR project, a European Commission’s Horizon 2020 research and innovation programme under grant agreement No 820621. The resulting software is released as part of LinkSmart, an open-source IoT prototyping platform.  

# References
