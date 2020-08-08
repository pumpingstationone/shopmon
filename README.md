# ShopMon

ShopMon is a system for showing occupancy at [Pumping Station: One](https://pumpingstationone.org) in real-time. The publicly-accessable front end to the system is [this web page](https://shopmon.pumpingstationone.org). The system uses passive IR sensors to detect motion and puts all the information on an MQTT topic which others can read.

Because the system uses passive IR sensors, it is not capable of determining _who_ is in a specific location, only that _someone_ is there. 

## Components
Each component has its own README for more details.
### Sensors
The code that reads the sensor data from the main panel and sends to an MQTT topic.

### SensorStatus
This is an intermediary program to allow other components (e.g. the website) to have more fine-grained control over sensor activity. 

### Website
All the code for [the public website](https://shopmon.pumpingstationone.org).