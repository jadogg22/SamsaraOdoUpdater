# Samsara Odometer Updater

## Overview

This project aims to bride the gap between the two programs used at our workplace: Samsara and Dossier. Samsara is our tractor's ELD(Electronic Logging Device) progam, while Dossier is using for managing truck repairs and maintence. The main goal of this program is to update the odometer readings in Dossier based on the data from Samsara.

## Features

* Retrieves odometer readings from Samsara for each vehicle.
* Retrieves odometer readings from Omnitracks (we havn't quite phased out omni)
* Updates each meter reading in Dosseir.

## Installation

1. Clone this repository to your local machine:
```bash
git clone https://github.com/jadogg22/SamsaraUpdater
```
2. Install any dependacies requred by the project:
```bash
cd SamsaraOdoUpdater
go mod tidy
```
3. build the project:
```bash
go build
```
## Usage 
1. Ensure you have valid authentication crededtials for Samsara, Omnitracks and Dossier. saved in a .env file

```
subscribersId=7
omniUsername=
omniPassword=

dossUsername=
dossPassword=
client_secret=

samsaraAccessToken=
```
2. Setup enviroment variables or provide the necessary credentials.
3. run the program:

```bash
./samsaraOdoUpdater
```

## Sample Output

```
Grabbing Odometer Readings from Samsara
/total assets XXX
Name: XXX
Timestamp 2024-05-10T14:47:17Z
Odometer: XXX,XXX
Elmwood Court Selden, NY 11784 Lat: 47.646956, Long: -191.908008
--------------------------

Name: XXXX
Timestamp 2024-05-08T23:18:13Z
Odometer: XXXX
9545 Cobblestone Ave. East Meadow, NY 11554 
--------------------------

Name: xxx
Timestamp 2021-09-21T12:11:04Z
Odometer: XXX,XXX
I 15, Malad, ID, 83252
--------------------------

Done!
Grabbing data from Omnitracks...
Removed 23 Duplacates

[=========>                                        ] (50/298)
```
## Highlevel overview

### Samsara part 

Getting the Odometers readings is actually very simple. As long as we have an api key we can make a quick and easy GET request to Samsaras "snapshot" endpoint and it sends us gps and odometer data for each asset in the fleet. By sending and 'Authorization' header with the 'bearer {samsaraAPIKey}' we have access to the endpoint. we add the type of data we want as a 'types' and add 'gps,obdOdometerMeters'. That returns a json package with all the data that we need. from there we just parse the data and put it in an array to be updated.

## Omnitracks

because we're phasing this out it doesnt super matter but i'll add it anyway. Basically this was pretty confusing to figure out but once you get it its fairly simple omnitracks uses an old soap api structure so you need to send envelopes to the endpoints. How its set up is every so often the Omnitracks elds send a bunch of data to their severs they put all of that into a data queue. 

In order to get the data we need to just ping it once with a throwaway transaction ID. Parse all of the data(its alot) you get like that "instants" data. for the last "transacton" of data you get you place that ID into the envenlope you send at it will continue where it left off.

Thats how it works at a high level.

Because we cant get just a snapshot of the data and we have to parse all of the data that we have its easiest to make a map/dictionary to the assetID and if that assetID is already in the map and the odometer is greater we just overwrite that data with the new data we have received untill there is no more data to receive from Omnitracks.

Now because I'm lazy (arn't we all) I didn't really wanna think bout this data too much because before long we wont be getting anything from omnitracks. So essentully I loop through the existing data from samsara and if it is also in the data map from Omnitracks I just delete it. This just leaves me with a map of data that is only in omnitracks. again because I'm kinda lazy I now loop through the map and append each tractor to the samsara vehicle data slice.

Then we're ready to update the Vehicles in Dossier

## Dossier

Because dossier kinda sucks there was no way(that I know of) to use our asset ID number to just update the odometer. This means that for each asset we have to make 2 get requests. fun...

after authenticating we can take our first asset we need to update and ask if it exists in their system. They then return a buncha junk with our asset ID and their equipmentID we need this equipmentID inorder to then create the json to make a odometer update.

Then I basically do that for all of the assets in the slice. I spent a lot of time trying to figure the best method out. Basically I reverse engenerred their requests for their ui and and it seems to be kinda what their doing. I've thought about caching the equipmentId's to make it quicker but honestly. I hate their system and there should be a bulk updater and or a way to just update them based on our asset ID. If their not going to provide the basic functionality like that they can pay for the 500 requests I make every day. Who knows what the other companies using their services do. Thanks for coming to my ted talk.



