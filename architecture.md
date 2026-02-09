# Tech Stack
- Go Gin
- MQTT
- Postgres
- RabbitMQ
- Docker


# ADT

```haskell
-- Core domain types
data Location = Location
  { lat       :: Double
  , lon       :: Double
  , timestamp :: Timestamptz
  } deriving (Show)

data Vehicle = Vehicle
  { vehicleId :: String  -- e.g. "B1234XYZ"
  } deriving (Show)

data VehicleLocation = VehicleLocation
  { vehicle   :: Vehicle
  , location  :: Location
  } deriving (Show)

-- MQTT inbound message (raw from topic)
data LocationMessage = LocationMessage
  { vehicleId :: String
  , latitude  :: Double
  , longitude :: Double

-- Geofence types
data GeoPoint = GeoPoint
  { lat    :: Double
  , lon    :: Double
  , radius :: Double  -- meters (e.g. 50)
  } deriving (Show)

data GeofenceEventType = GeofenceEntry | GeofenceExit
  deriving (Show)

data GeofenceAlert = GeofenceAlert
  { vehicleId :: String
  , event     :: GeofenceEventType
  , location  :: Location
  , geofence  :: GeoPoint
  , timestamp :: Timestamptz
  } deriving (Show)

-- API response types
data LocationResponse = LocationResponse
  { vehicleId :: String
  , latitude  :: Double
  , longitude :: Double
  , timestamp :: Int64
  } deriving (Show)

data HistoryQuery = HistoryQuery
  { vehicleId :: String
  , start     :: Int64
  , end       :: Int64
  } deriving (Show)

-- Validation
data ValidationError
  = InvalidCoordinate String
  | MissingField String
  | InvalidTimestamp
  deriving (Show)
```
