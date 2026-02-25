extends Node

signal login_success
signal login_failed(error: String)
signal register_success
signal register_failed(error: String)

var token: String = ""
var user_id: int = 0
var username: String = ""

var is_logged_in: bool:
	get:
		return token != ""

func register(user: String, password: String):
	var url = NetworkConfig.base_url + "/api/auth/register"
	var body = JSON.stringify({"username": user, "password": password})
	var http = HTTPRequest.new()
	add_child(http)
	http.request_completed.connect(_on_register_completed.bind(http))
	http.request(url, ["Content-Type: application/json"], HTTPClient.METHOD_POST, body)

func login(user: String, password: String):
	var url = NetworkConfig.base_url + "/api/auth/login"
	var body = JSON.stringify({"username": user, "password": password})
	var http = HTTPRequest.new()
	add_child(http)
	http.request_completed.connect(_on_login_completed.bind(http))
	http.request(url, ["Content-Type: application/json"], HTTPClient.METHOD_POST, body)

func _on_register_completed(result: int, response_code: int, _headers: PackedStringArray, body: PackedByteArray, http: HTTPRequest):
	http.queue_free()
	if result != HTTPRequest.RESULT_SUCCESS:
		register_failed.emit("Connection failed")
		return
	var json = JSON.new()
	json.parse(body.get_string_from_utf8())
	var data = json.get_data()
	if response_code == 201:
		token = data.get("token", "")
		user_id = data.get("user_id", 0)
		username = data.get("username", "")
		register_success.emit()
	else:
		register_failed.emit(data.get("error", "Registration failed"))

func _on_login_completed(result: int, response_code: int, _headers: PackedStringArray, body: PackedByteArray, http: HTTPRequest):
	http.queue_free()
	if result != HTTPRequest.RESULT_SUCCESS:
		login_failed.emit("Connection failed")
		return
	var json = JSON.new()
	json.parse(body.get_string_from_utf8())
	var data = json.get_data()
	if response_code == 200:
		token = data.get("token", "")
		user_id = data.get("user_id", 0)
		username = data.get("username", "")
		login_success.emit()
	else:
		login_failed.emit(data.get("error", "Login failed"))

func logout():
	token = ""
	user_id = 0
	username = ""
	WebSocketClient.disconnect_from_server()

func get_auth_headers() -> PackedStringArray:
	return ["Authorization: Bearer " + token, "Content-Type: application/json"]
