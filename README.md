[![CI](https://github.com/armsnyder/a2s-exporter/actions/workflows/ci.yaml/badge.svg)](https://github.com/armsnyder/a2s-exporter/actions/workflows/ci.yaml)

# [A2S Exporter](https://github.com/armsnyder/a2s-exporter)

A Prometheus exporter for Steam game server info.

Supports all Steam game servers which speak the UDP-based A2S query protocol, for example:

* Counter-Strike
* The Forest
* Rust
* Team Fortress 2
* Valheim

## Usage

The image is hosted on Docker Hub. ([Link](https://hub.docker.com/r/armsnyder/a2s-exporter))

```
docker run --rm -p 9841:9841 armsnyder/a2s-exporter --address myserver.example.com:12345
```

### Arguments

Arguments may be provided using commandline flags or environment variables.

#### Required

Flag | Variable | Help
--- | --- | ---
--address | A2S_EXPORTER_QUERY_ADDRESS | Address of the A2S query server as host:port (This is a separate port from the main server port).

#### Optional

Flag | Variable | Default | Help
--- | --- | --- | ---
--port | A2S_EXPORTER_PORT | 9841 | Port for the metrics exporter.
--path | A2S_EXPORTER_PATH | /metrics | Path for the metrics exporter.
--namespace | A2S_EXPORTER_NAMESPACE | a2s | Namespace prefix for all exported a2s metrics.
--exclude-player-metrics | A2S_EXPORTER_EXCLUDE_PLAYER_METRICS | false | If true, exclude all `player_*` metrics. This option may be necessary for some servers.
--a2s-only-metrics | A2S_EXPORTER_A2S_ONLY_METRICS | false | If true, excludes Go runtime and promhttp metrics.
--max-packet-size | A2S_EXPORTER_MAX_PACKET_SIZE | 1400 | Advanced option to set a non-standard max packet size of the A2S query server.

#### Special

Flag | Help
--- | ---
-h | Show help.
--version | Show build version.

## Exported Metrics

Metrics names are prefixed with a namespace (default `a2s_`).

Name | Help | Labels
--- | --- | ---
player_count | Total number of connected players. | server_name
player_duration | Time (in seconds) player has been connected to the server. | server_name player_name player_index
player_score | Player's score (usually \"frags\" or \"kills\"). | server_name player_name player_index
player_the_ship_deaths | Player's deaths in a The Ship server. | server_name player_name player_index
player_the_ship_money | Player's money in a The Ship server. | server_name player_name player_index
player_up | Was the last player info query successful. |
server_bots | Number of bots on the server. | server_name
server_info | Non-numerical server info, including server_steam_id and version. The value is 1, and info is in the labels. | server_name map folder game server_type server_os version server_id keywords server_game_id server_steam_id the_ship_mode source_tv_name
server_max_players | Maximum number of players the server reports it can hold. | server_name
server_players | Number of players on the server. | server_name
server_port | The server's game port number. | server_name
server_protocol | Protocol version used by the server. | server_name
server_source_tv_port | Spectator port number for SourceTV. | server_name
server_the_ship_duration | Time (in seconds) before a player is arrested while being witnessed in a The Ship server. | server_name
server_the_ship_witnesses | The number of witnesses necessary to have a player arrested in a The Ship server. | server_name
server_up | Was the last server info query successful. |
server_vac | Specifies whether the server uses VAC (0 for unsecured, 1 for secured). | server_name
server_visibility | Indicates whether the server requires a password (0 for public, 1 for private). | server_name

## Credits

This exporter depends on [rumblefrog/go-a2s](https://github.com/rumblefrog/go-a2s) (MIT). Big thanks to them!
