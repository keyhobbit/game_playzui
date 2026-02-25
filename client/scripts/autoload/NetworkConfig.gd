extends Node

var server_host: String = "51.79.138.246"
var server_port: int = 8700
var use_ssl: bool = false

var base_url: String:
	get:
		var protocol = "https" if use_ssl else "http"
		return "%s://%s:%d" % [protocol, server_host, server_port]

var ws_url: String:
	get:
		var protocol = "wss" if use_ssl else "ws"
		return "%s://%s:%d/ws" % [protocol, server_host, server_port]

func _ready():
	var args = OS.get_cmdline_args()
	for arg in args:
		if arg.begins_with("--server="):
			server_host = arg.split("=")[1]
		elif arg.begins_with("--port="):
			server_port = int(arg.split("=")[1])
		elif arg == "--ssl":
			use_ssl = true
