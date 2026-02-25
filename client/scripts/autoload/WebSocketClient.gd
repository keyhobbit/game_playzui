extends Node

signal connected
signal disconnected
signal message_received(msg: Dictionary)
signal connection_error(error: String)

var _socket: WebSocketPeer = null
var _connected: bool = false
var _reconnect_timer: float = 0.0
var _reconnect_delay: float = 3.0
var _should_reconnect: bool = false

func connect_to_server():
	if _connected:
		return
	if not AuthManager.is_logged_in:
		connection_error.emit("Not logged in")
		return

	_socket = WebSocketPeer.new()
	var url = NetworkConfig.ws_url + "?token=" + AuthManager.token
	var err = _socket.connect_to_url(url)
	if err != OK:
		connection_error.emit("Failed to initiate connection")
		_socket = null
		return
	_should_reconnect = true

func disconnect_from_server():
	_should_reconnect = false
	if _socket:
		_socket.close()
		_socket = null
	_connected = false

func send_message(type: String, payload: Dictionary = {}):
	if not _connected or _socket == null:
		return
	var msg = JSON.stringify({"type": type, "payload": payload})
	_socket.send_text(msg)

func join_room(room_id: int):
	send_message("join_room", {"room_id": room_id})

func leave_room():
	send_message("leave_room", {})

func set_ready():
	send_message("ready", {})

func play_cards(cards: Array):
	send_message("play_cards", {"cards": cards})

func pass_turn():
	send_message("pass_turn", {})

func send_chat(message: String):
	send_message("chat", {"message": message})

func auto_match(ante_level: int):
	send_message("auto_match", {"ante_level": ante_level})

func _process(delta: float):
	if _socket == null:
		if _should_reconnect and not _connected:
			_reconnect_timer += delta
			if _reconnect_timer >= _reconnect_delay:
				_reconnect_timer = 0.0
				connect_to_server()
		return

	_socket.poll()
	var state = _socket.get_ready_state()

	match state:
		WebSocketPeer.STATE_OPEN:
			if not _connected:
				_connected = true
				_reconnect_timer = 0.0
				connected.emit()
			while _socket.get_available_packet_count() > 0:
				var packet = _socket.get_packet()
				var text = packet.get_string_from_utf8()
				var json = JSON.new()
				var err = json.parse(text)
				if err == OK:
					message_received.emit(json.get_data())
		WebSocketPeer.STATE_CLOSING:
			pass
		WebSocketPeer.STATE_CLOSED:
			var was_connected = _connected
			_connected = false
			_socket = null
			if was_connected:
				disconnected.emit()
