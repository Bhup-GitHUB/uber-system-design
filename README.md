# Uber System Design - Geospatial Indexing

Learning Uber backend and scalability, specifically focusing on the **Geospatial Indexing** system for real-time driver location tracking and management.

## Overview

Uber's geospatial indexing system is designed to handle millions of driver location updates in real-time, enabling efficient spatial queries for ride matching, driver discovery, and location-based services. The system processes location data through multiple layers of validation, indexing, and caching to provide low-latency access to driver locations.

## Architecture Diagrams

The system architecture is illustrated in two complementary diagrams:

- **Diagram 1** (`1.png`): Shows the initial data ingestion flow from driver applications through load balancing, API gateway, and into Kafka queues with city-based partitioning.
- **Diagram 2** (`2.png`): Provides a detailed view of the complete pipeline including stream processing, geo-routing, spatial indexing, caching, and monitoring.

## System Architecture

### High-Level Flow

```
Driver Apps → Load Balancer → API Gateway → Location Services → Kafka → 
Stream Processing → Geo-Routing → Spatial Indexes → Redis Cache → In-Memory State
```

## Component Breakdown

### 1. Client Layer

**Driver Applications**
- Multiple driver app instances send location updates
- Each update includes: driver ID, latitude, longitude, timestamp, and metadata

### 2. Load Balancing Layer

**Nginx Load Balancer**
- Distributes incoming requests across multiple backend services
- Provides high availability with a backup load balancer for failover
- Handles SSL termination and request routing

**Features:**
- Primary and backup load balancer configuration
- Health checks and automatic failover
- Request distribution based on geographic proximity or load

### 3. Gateway Layer

**API Gateway**
- Single entry point for all location update requests
- Routes requests to appropriate microservices

**Rate Limiter**
- Protects backend services from traffic spikes
- Implements per-driver rate limiting
- Prevents abuse and ensures fair resource allocation

**JWT Authentication**
- Validates driver identity using JSON Web Tokens
- Ensures only authenticated drivers can update their location
- Provides security and prevents unauthorized access

### 4. Location Services Layer

**Location Services (Multiple Instances)**
- Horizontally scalable microservices handling location updates
- Each service instance processes location updates independently
- Performs initial validation and data normalization

**Responsibilities:**
- Receive authenticated location updates
- Validate coordinate ranges and data format
- Normalize location data
- Publish to Kafka queue

### 5. Message Queue Layer

**Kafka Queue: "driver location updates" Topic**

**City-Based Partitioning:**
- **Partition Bengaluru**: Handles location updates for drivers in Bengaluru
- **Partition Delhi**: Handles location updates for drivers in Delhi
- **Partition Mumbai**: Handles location updates for drivers in Mumbai

**Benefits of City-Based Partitioning:**
- **Geographic Isolation**: Each city's data is processed independently
- **Parallel Processing**: Multiple cities can be processed simultaneously
- **Scalability**: Easy to add new cities by adding new partitions
- **Fault Isolation**: Issues in one city don't affect others
- **Data Locality**: Related queries are likely to access the same partition

### 6. Stream Processing Layer

**Consumer Group**
- Consumes messages from Kafka partitions
- Processes location updates in real-time

**Processing Steps:**

1. **Location Validation**
   - Validates coordinate accuracy
   - Checks for reasonable movement speeds
   - Filters out invalid or duplicate updates
   - Ensures data quality before indexing

2. **Anomaly Detection**
   - Detects unusual patterns (e.g., impossible speeds, teleportation)
   - Identifies GPS errors or malicious updates
   - Flags suspicious location data for review
   - Ensures data integrity

### 7. Storage Layer

**Cassandra - Location History**
- Stores historical location data for analytics
- Time-series data storage for driver movement patterns
- Supports queries like "where was driver X at time Y"
- Used for fraud detection and analytics

**Time Series Metrics**
- Stores system performance metrics
- Tracks processing latency, throughput, and error rates
- Used for monitoring and alerting

### 8. Monitoring Layer

**Prometheus**
- Collects metrics from all system components
- Time-series database for metrics storage
- Provides querying capabilities for metrics

**Grafana**
- Visualizes metrics and system health
- Real-time dashboards for operations team
- Performance monitoring and trend analysis

**Alerts**
- Configurable alerting based on metrics thresholds
- Notifies on system anomalies or performance degradation
- Ensures proactive issue detection

### 9. Geo-Routing Layer

**Geo Router (City-Based)**
- Routes validated location updates to city-specific indexing systems
- Determines which city a location belongs to
- Distributes load across city-specific indexers
- Ensures efficient spatial data organization

### 10. Spatial Indexing Layer

Each city maintains three types of spatial indexes:

**For Each City (Bengaluru, Delhi, Mumbai):**

1. **Quad Tree Index**
   - Hierarchical tree structure dividing space into quadrants
   - Efficient for point queries and range searches
   - Good for uniform distribution of drivers
   - O(log n) query time

2. **Grid Index**
   - Divides geographic area into fixed-size grid cells
   - Simple and fast for nearby driver searches
   - Excellent for uniform grid-based queries
   - O(1) cell lookup, O(k) where k is drivers in cell

3. **S2 Index**
   - Google's S2 geometry library for spherical geometry
   - Handles Earth's curvature accurately
   - Hierarchical cell-based indexing
   - Excellent for global-scale queries
   - Supports efficient nearest neighbor searches

**Why Multiple Indexes?**
- Different indexes optimize for different query patterns
- Quad Tree: Good for hierarchical queries
- Grid: Fast for nearby searches in dense areas
- S2: Best for accurate geographic calculations
- Query router selects optimal index based on query type

### 11. Caching Layer

**Redis Shards (Per City)**
- Each city has its own Redis shard
- Caches frequently accessed driver locations
- Reduces load on spatial indexes
- Provides sub-millisecond read latency

**Benefits:**
- **Low Latency**: In-memory access for hot data
- **Scalability**: Sharding by city distributes load
- **High Availability**: Redis cluster with replication

### 12. In-Memory Driver State

**Centralized Driver State**
- Aggregates data from all Redis shards
- Maintains current state of all active drivers
- Provides unified view for ride matching

**Outputs:**
- **Active Drivers**: List of currently available drivers
- **Driver Metadata**: Additional information (rating, vehicle type, etc.)

## Data Flow Example

1. **Driver Update**: Driver app sends location update (lat: 12.9716, lng: 77.5946)
2. **Load Balancing**: Request routed through Nginx load balancer
3. **Authentication**: JWT token validated at API gateway
4. **Rate Limiting**: Request checked against rate limits
5. **Location Service**: Update received and normalized
6. **Kafka**: Published to "Bengaluru" partition (based on coordinates)
7. **Stream Processing**: 
   - Location validated (coordinates in valid range)
   - Anomaly detection (speed reasonable, no teleportation)
8. **Geo-Routing**: Routed to Bengaluru indexing system
9. **Spatial Indexing**: Updated in Quad Tree, Grid, and S2 indexes
10. **Redis Cache**: Cached in Bengaluru Redis shard
11. **In-Memory State**: Updated in centralized driver state
12. **Storage**: Historical data written to Cassandra

## Design Decisions & Scalability

### Why City-Based Partitioning?

1. **Geographic Locality**: Most queries are city-specific
2. **Parallel Processing**: Independent processing per city
3. **Horizontal Scaling**: Easy to add new cities
4. **Fault Isolation**: City-level failure isolation
5. **Data Locality**: Related data stays together

### Why Multiple Spatial Indexes?

1. **Query Optimization**: Different indexes for different query types
2. **Performance**: Trade-offs between update speed and query speed
3. **Accuracy**: S2 handles Earth's curvature for global accuracy
4. **Flexibility**: Can route queries to best-performing index

### Why Kafka?

1. **High Throughput**: Handles millions of messages per second
2. **Durability**: Messages persisted and replicated
3. **Scalability**: Easy to add partitions and consumers
4. **Decoupling**: Producers and consumers are independent
5. **Replay Capability**: Can reprocess messages if needed

### Why Redis for Caching?

1. **Low Latency**: Sub-millisecond read times
2. **High Throughput**: Handles millions of operations per second
3. **Data Structures**: Supports complex data types
4. **Persistence**: Optional persistence for durability
5. **Sharding**: Easy horizontal scaling

### Scalability Considerations

- **Horizontal Scaling**: All components can scale horizontally
- **Partitioning**: City-based partitioning enables independent scaling
- **Caching**: Redis reduces load on expensive spatial queries
- **Async Processing**: Kafka enables asynchronous processing
- **Load Balancing**: Distributes load across service instances

## Performance Characteristics

- **Latency**: Sub-100ms for location updates end-to-end
- **Throughput**: Millions of location updates per second
- **Query Performance**: Sub-10ms for nearby driver queries
- **Availability**: 99.99% uptime with redundancy at every layer

## Use Cases

1. **Ride Matching**: Find nearest available drivers to rider location
2. **Surge Pricing**: Calculate demand based on driver density
3. **ETA Calculation**: Estimate arrival time based on driver locations
4. **Fraud Detection**: Detect unusual location patterns
5. **Analytics**: Historical location data for business insights

## Conclusion

Uber's geospatial indexing system demonstrates a sophisticated approach to handling real-time location data at scale. By combining city-based partitioning, multiple spatial indexes, distributed caching, and stream processing, the system achieves both high throughput and low latency while maintaining data quality and system reliability.
