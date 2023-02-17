# ShopMonBot
## Purpose
`ShopMonBot` is a Slack-based bot that will tell you when someone was last in an area. It can be a specific area (e.g. `!area woodshop`) or a report on all the currently known areas (`!area all`).

## Design
The bot is written in Go and makes use of the [nlopes/slack](https://github.com/nlopes/slack) and [eclipse/paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) libraries for Slack and MQTT, respectively. 

It listens on the MQTT topic and for each message it receives, checks the `sensorMap`, a K/V store of area name (string) to last seen (timestamp) and if there is already an entry, updates it with the new timestamp, otherwise simply adds it. By keeping the insertion dynamic and not depending on pre-determined fixed entries, new areas can be brought online without requiring the bot to be restarted or in any way updated; it will simply add those new areas as it sees them.

When someone invokes the bot with `!area <area name>` or `!area all`, it will go through the map and check if the the key matches the requested area. While the whole point of a map is fast searching, we are in fact rolling through it like a list or an array. The reason for this is that, in this general context, there are relatively few areas (think less than a dozen entries) _and_ we want to build a list of known areas in case the person specifically asked for something we don't (yet) know about. This way we can return a full list of the areas for the user to choose from. It's also necessary in the event someone asks for all the areas, which in practice turns out to be the more popular option. 
