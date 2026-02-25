extends Control

@onready var username_label: Label = %UsernameLabel
@onready var gold_label: Label = %GoldLabel
@onready var room_list: VBoxContainer = %RoomList
@onready var status_label: Label = %StatusLabel

var rooms: Array = []
var current_filter: int = 0

func _ready():
	username_label.text = AuthManager.username
	gold_label.text = "Gold: ---"
	WebSocketClient.message_received.connect(_on_ws_message)
	WebSocketClient.connected.connect(_on_ws_connected)
	WebSocketClient.disconnected.connect(_on_ws_disconnected)
	_fetch_profile()
	if WebSocketClient._connected:
		_on_ws_connected()

func _fetch_profile():
	var url = NetworkConfig.base_url + "/api/user/profile"
	var http = HTTPRequest.new()
	add_child(http)
	http.request_completed.connect(_on_profile_loaded.bind(http))
	http.request(url, AuthManager.get_auth_headers(), HTTPClient.METHOD_GET)

func _on_profile_loaded(_result: int, response_code: int, _headers: PackedStringArray, body: PackedByteArray, http: HTTPRequest):
	http.queue_free()
	if response_code == 200:
		var json = JSON.new()
		json.parse(body.get_string_from_utf8())
		var data = json.get_data()
		gold_label.text = "Gold: %d" % data.get("gold_balance", 0)

func _on_ws_connected():
	status_label.text = "Connected"
	_fetch_rooms()

func _on_ws_disconnected():
	status_label.text = "Disconnected - reconnecting..."

func _fetch_rooms():
	var url = NetworkConfig.base_url + "/api/rooms?all=true"
	if current_filter > 0:
		url += "&ante=%d" % current_filter
	var http = HTTPRequest.new()
	add_child(http)
	http.request_completed.connect(_on_rooms_loaded.bind(http))
	http.request(url, AuthManager.get_auth_headers(), HTTPClient.METHOD_GET)

func _on_rooms_loaded(_result: int, response_code: int, _headers: PackedStringArray, body: PackedByteArray, http: HTTPRequest):
	http.queue_free()
	if response_code == 200:
		var json = JSON.new()
		json.parse(body.get_string_from_utf8())
		var data = json.get_data()
		rooms = data.get("rooms", [])
		_render_room_list()

func _render_room_list():
	for child in room_list.get_children():
		child.queue_free()

	for room in rooms:
		var btn = Button.new()
		btn.text = "%s  |  %d/4 players  |  %s" % [
			room.get("name", "Room"),
			room.get("player_count", 0),
			room.get("phase", "LOBBY")
		]
		btn.custom_minimum_size = Vector2(0, 50)
		var room_id = room.get("id", 0)
		btn.pressed.connect(_on_room_join.bind(room_id))
		room_list.add_child(btn)

func _on_room_join(room_id: int):
	WebSocketClient.join_room(room_id)

func _on_ws_message(msg: Dictionary):
	var type = msg.get("type", "")
	match type:
		"match_found":
			var payload = msg.get("payload", {})
			status_label.text = "Match found! Joining room..."
			get_tree().change_scene_to_file("res://scenes/game_room/game_room.tscn")
		"room_update":
			_fetch_rooms()
		"card_dealt", "game_state":
			get_tree().change_scene_to_file("res://scenes/game_room/game_room.tscn")
		"error":
			var payload = msg.get("payload", {})
			status_label.text = payload.get("error", "Unknown error")

func _on_logout_pressed():
	AuthManager.logout()
	get_tree().change_scene_to_file("res://scenes/login/login.tscn")

func _on_filter_all():
	current_filter = 0
	_fetch_rooms()

func _on_filter_100():
	current_filter = 100
	_fetch_rooms()

func _on_filter_500():
	current_filter = 500
	_fetch_rooms()

func _on_filter_1000():
	current_filter = 1000
	_fetch_rooms()

func _on_auto_match_100():
	WebSocketClient.auto_match(100)
	status_label.text = "Finding match (100G)..."

func _on_auto_match_500():
	WebSocketClient.auto_match(500)
	status_label.text = "Finding match (500G)..."

func _on_auto_match_1000():
	WebSocketClient.auto_match(1000)
	status_label.text = "Finding match (1000G)..."
