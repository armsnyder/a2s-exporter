package collector

import (
	"fmt"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rumblefrog/go-a2s"
)

type Collector struct {
	addr                 string
	clientOptions        []func(*a2s.Client) error
	excludePlayerMetrics bool
	client               *a2s.Client
	descs                map[string]*prometheus.Desc
}

type adder func(name string, value float64, labelValues ...string)

func New(namespace, addr string, excludePlayerMetrics bool, clientOptions ...func(*a2s.Client) error) *Collector {
	descs := make(map[string]*prometheus.Desc)

	fullDesc := func(name, help string, labels ...string) {
		descs[name] = prometheus.NewDesc(prometheus.BuildFQName(namespace, "", name), help, labels, nil)
	}
	basicDesc := func(name string, help string) {
		fullDesc(name, help, "server_name")
	}
	playerDesc := func(name string, help string) {
		fullDesc(name, help, "server_name", "player_name", "player_index")
	}

	fullDesc("server_info", "Non-numerical server info, including server_steam_id and version. The value is 1, and info is in the labels.",
		"server_name", "map", "folder", "game", "server_type", "server_os", "version", "server_id", "keywords", "server_game_id", "server_steam_id", "the_ship_mode", "source_tv_name")

	fullDesc("server_up", "Was the last server info query successful.")
	fullDesc("player_up", "Was the last player info query successful.")

	basicDesc("server_protocol", "Protocol version used by the server.")
	basicDesc("server_players", "Number of players on the server.")
	basicDesc("server_max_players", "Maximum number of players the server reports it can hold.")
	basicDesc("server_bots", "Number of bots on the server.")
	basicDesc("server_visibility", "Indicates whether the server requires a password (0 for public, 1 for private).")
	basicDesc("server_vac", "Specifies whether the server uses VAC (0 for unsecured, 1 for secured).")
	basicDesc("server_port", "The server's game port number.")
	basicDesc("server_source_tv_port", "Spectator port number for SourceTV.")
	basicDesc("server_the_ship_witnesses", "The number of witnesses necessary to have a player arrested in a The Ship server.")
	basicDesc("server_the_ship_duration", "Time (in seconds) before a player is arrested while being witnessed in a The Ship server.")

	basicDesc("player_count", "Total number of connected players.")
	playerDesc("player_duration", "Time (in seconds) player has been connected to the server.")
	playerDesc("player_score", `Player's score (usually "frags" or "kills").`)
	playerDesc("player_the_ship_deaths", "Player's deaths in a The Ship server.")
	playerDesc("player_the_ship_money", "Player's money in a The Ship server.")

	return &Collector{
		addr:                 addr,
		clientOptions:        clientOptions,
		excludePlayerMetrics: excludePlayerMetrics,
		descs:                descs,
	}
}

func (c *Collector) Describe(descs chan<- *prometheus.Desc) {
	for _, desc := range c.descs {
		descs <- desc
	}
}

func (c *Collector) Collect(metrics chan<- prometheus.Metric) {
	serverInfo, playerInfo := c.queryInfo(c.excludePlayerMetrics)

	truthyFloat := func(v interface{}) float64 {
		if reflect.ValueOf(v).IsNil() {
			return 0
		}
		return 1
	}

	add := func(name string, value float64, labelValues ...string) {
		metrics <- prometheus.MustNewConstMetric(c.descs[name], prometheus.GaugeValue, value, labelValues...)
	}

	add("server_up", truthyFloat(serverInfo))

	if !c.excludePlayerMetrics {
		add("player_up", truthyFloat(playerInfo))
	}

	addPreLabelled := func(name string, value float64, labelValues ...string) {
		labelValues2 := []string{serverInfo.Name}
		labelValues2 = append(labelValues2, labelValues...)
		add(name, value, labelValues2...)
	}

	c.collectServerInfo(serverInfo, addPreLabelled)
	c.collectPlayerInfo(playerInfo, addPreLabelled)
}

// queryInfo queries the A2S server over UDP. Failure will result in one or both of the return values being nil.
func (c *Collector) queryInfo(excludePlayerMetrics bool) (serverInfo *a2s.ServerInfo, playerInfo *a2s.PlayerInfo) {
	var err error

	// Lazy initialization of UDP client.
	if c.client == nil {
		c.client, err = a2s.NewClient(c.addr, c.clientOptions...)
		if err != nil {
			fmt.Println("Could not create A2S client:", err)
			return
		}
	}

	// Query server info.
	serverInfo, err = c.client.QueryInfo()
	if err != nil {
		fmt.Println("Could not query server info:", err)
		return
	}

	if excludePlayerMetrics {
		return
	}

	// A quirk of the a2s-go client is that in order for The Ship player queries to succeed, the client must be
	// constructed with The Ship App ID.
	playerClient := c.client
	if a2s.AppID(serverInfo.ID) == a2s.App_TheShip {
		options := []func(*a2s.Client) error{a2s.SetAppID(int32(serverInfo.ID))}
		options = append(options, c.clientOptions...)
		playerClient, err = a2s.NewClient(c.addr, options...)
		if err != nil {
			fmt.Println("Could not create A2S client for The Ship player query:", err)
			return
		}
	}

	// Query player info.
	// SourceTV does not respond to player queries.
	if serverInfo.ServerType != a2s.ServerType_SourceTV {
		playerInfo, err = playerClient.QueryPlayer()
		if err != nil {
			fmt.Println("Could not query player info:", err)
			return
		}
	}

	return
}

func (c *Collector) collectServerInfo(serverInfo *a2s.ServerInfo, add adder) {
	if serverInfo == nil {
		return
	}

	nilSafe := func(check interface{}, do func() string) string {
		if reflect.ValueOf(check).IsNil() {
			return ""
		}
		return do()
	}

	add("server_info", 1,
		serverInfo.Map,
		serverInfo.Folder,
		serverInfo.Game,
		serverInfo.ServerType.String(),
		serverInfo.ServerOS.String(),
		serverInfo.Version,
		fmt.Sprintf("%d", serverInfo.ID),
		nilSafe(serverInfo.ExtendedServerInfo, func() string { return serverInfo.ExtendedServerInfo.Keywords }),
		nilSafe(serverInfo.ExtendedServerInfo, func() string { return fmt.Sprintf("%d", serverInfo.ExtendedServerInfo.GameID) }),
		nilSafe(serverInfo.ExtendedServerInfo, func() string { return fmt.Sprintf("%d", serverInfo.ExtendedServerInfo.SteamID) }),
		nilSafe(serverInfo.TheShip, func() string { return serverInfo.TheShip.Mode.String() }),
		nilSafe(serverInfo.SourceTV, func() string { return serverInfo.SourceTV.Name }),
	)

	addPos := func(name string, value float64, labelValues ...string) {
		if value <= 0 {
			return
		}
		add(name, value, labelValues...)
	}

	addBool := func(name string, value bool, labelValues ...string) {
		var asFloat float64
		if value {
			asFloat = 1
		}
		add(name, asFloat, labelValues...)
	}

	addPos("server_protocol", float64(serverInfo.Protocol))
	add("server_players", float64(serverInfo.Players))
	add("server_max_players", float64(serverInfo.MaxPlayers))
	add("server_bots", float64(serverInfo.Bots))
	addBool("server_visibility", serverInfo.Visibility)
	addBool("server_vac", serverInfo.VAC)

	if serverInfo.ExtendedServerInfo != nil {
		addPos("server_port", float64(serverInfo.ExtendedServerInfo.Port))
	}

	if serverInfo.SourceTV != nil {
		addPos("server_source_tv_port", float64(serverInfo.SourceTV.Port))
	}

	if serverInfo.TheShip != nil {
		add("server_the_ship_witnesses", float64(serverInfo.TheShip.Witnesses))
		add("server_the_ship_duration", float64(serverInfo.TheShip.Duration))
	}
}

func (c *Collector) collectPlayerInfo(playerInfo *a2s.PlayerInfo, add adder) {
	if playerInfo == nil {
		return
	}

	add("player_count", float64(playerInfo.Count))

	for _, player := range c.uniquePlayers(playerInfo.Players) {
		labelValues := []string{player.Name, fmt.Sprintf("%d", player.Index)}

		add("player_duration", float64(player.Duration), labelValues...)
		add("player_score", float64(player.Score), labelValues...)

		if player.TheShip != nil {
			add("player_the_ship_deaths", float64(player.TheShip.Deaths), labelValues...)
			add("player_the_ship_money", float64(player.TheShip.Money), labelValues...)
		}
	}
}

func (c *Collector) uniquePlayers(players []*a2s.Player) []*a2s.Player {
	// Some servers like Rust will assign a pool of random player names, which may contain duplicates
	// and cause errors in the Prometheus registry.

	result := make([]*a2s.Player, 0, len(players))
	seen := make(map[string]struct{})

	for _, player := range players {
		if _, ok := seen[player.Name]; ok {
			continue
		}

		seen[player.Name] = struct{}{}
		result = append(result, player)
	}

	return result
}
