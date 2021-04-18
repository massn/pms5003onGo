# pms5003onGo
A library of "Plantower PMS 5003 Particulate Sensor" written in Go.


## Usage
```bash
$ go build -o pms5003onGo
$ sudo ./pms5003onGo
+--------------------+--------+--------+
|        DATA        | NUMBER |  UNIT  |
+--------------------+--------+--------+
| PM1.0              |      0 | ug/m^3 |
| PM2.5              |      0 | ug/m^3 |
| PM10               |      0 | ug/m^3 |
| PM1.0 in atmos env |      0 | ug/m^3 |
| PM2.5 in atmos env |      0 | ug/m^3 |
| PM10 in atmos env  |      0 | ug/m^3 |
| 0.3um              |    222 | 1/0.1L |
| 0.5um              |     41 | 1/0.1L |
| 1.0um              |      1 | 1/0.1L |
| 2.5um              |      0 | 1/0.1L |
| 5.0um              |      0 | 1/0.1L |
| 10um               |      0 | 1/0.1L |
+--------------------+--------+--------+
```

## TODOs
- refactor log uses
- server mode
- refactor
