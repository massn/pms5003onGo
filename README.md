# pms5003onGo
A library for "Plantower PMS 5003 Particulate Sensor" written in Go.

## Usage
```bash
$ go build -o pms5003onGo
$ ./pms5003onGo --help
Usage of ./pms5003onGo:
  -d string
        device port to read. (default "/dev/ttyAMA0")
  -j    json output.
  -p string
        server port. (default "8080")
  -s    server mode.
  -t int
        timeout seconds. (default 5)
  -v    verbose output.
$ ./pms5003onGo -j
{
    "pm1p0": 2,
    "pm2p5": 2,
    "pm10": 3,
    "pm1p0atmos": 2,
    "pm2p5atmos": 2,
    "pm10atmos": 3,
    "dia0p3um": 522,
    "dia0p5um": 131,
    "dia1p0um": 14,
    "dia2p5um": 1,
    "dia5p0um": 1,
    "dia10um": 0,
    "Err": null
}
$ ./pms5003onGo
+--------------------+--------+--------+
|        DATA        | NUMBER |  UNIT  |
+--------------------+--------+--------+
| PM1.0              |      2 | ug/m^3 |
| PM2.5              |      2 | ug/m^3 |
| PM10               |      3 | ug/m^3 |
| PM1.0 in atmos env |      2 | ug/m^3 |
| PM2.5 in atmos env |      2 | ug/m^3 |
| PM10 in atmos env  |      3 | ug/m^3 |
| 0.3um              |    522 | 1/0.1L |
| 0.5um              |    131 | 1/0.1L |
| 1.0um              |     14 | 1/0.1L |
| 2.5um              |      1 | 1/0.1L |
| 5.0um              |      1 | 1/0.1L |
| 10um               |      0 | 1/0.1L |
+--------------------+--------+--------+
```