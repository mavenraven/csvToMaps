# CSV to Maps

## Goal

Users take their dog for a walk every day. These walks are tracked using GPS.


The [dataset](https://raw.githubusercontent.com/mavenraven/csvToMaps/master/src/test/resources/gps_dataset.csv) contains
raw GPS coordinates from a few different walks around NYC.



## Usage
```
Usage: CSVToMaps [options]
  Options:
  * --csvFileLocation
      Location of input GPS CSV data.
  * --mapboxAccessToken
      See https://docs.mapbox.com/help/how-mapbox-works/access-tokens/.
```
## Building
`./mvnw package`

## Running
`java -jar target/csvToMaps-1.0-SNAPSHOT.jar`
## Running Integration Tests
` ./mvnw verify -DmapboxAccessToken=<access token> -DdeepAIApiKey=<api key>`

The mapbox access token is passed through to the system under test and is used for map generation. One can be acquired at [https://docs.mapbox.com/help/how-mapbox-works/access-tokens/](https://docs.mapbox.com/help/how-mapbox-works/access-tokens/).

The deep AI API key is used to compare the output of the integration test with a previously generated map. The endpoint uses machine learning to compare how similar the two images are. A key can be acquired at [https://deepai.org/](https://deepai.org/).

