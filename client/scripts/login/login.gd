extends Control

@onready var username_input: LineEdit = %UsernameInput
@onready var password_input: LineEdit = %PasswordInput
@onready var login_button: Button = %LoginButton
@onready var register_button: Button = %RegisterButton
@onready var status_label: Label = %StatusLabel

func _ready():
	AuthManager.login_success.connect(_on_login_success)
	AuthManager.login_failed.connect(_on_login_failed)
	AuthManager.register_success.connect(_on_register_success)
	AuthManager.register_failed.connect(_on_register_failed)

func _on_login_pressed():
	var user = username_input.text.strip_edges()
	var pass = password_input.text
	if user.length() < 3 or pass.length() < 6:
		status_label.text = "Username (3+) and password (6+) required"
		return
	_set_buttons_disabled(true)
	status_label.text = "Logging in..."
	AuthManager.login(user, pass)

func _on_register_pressed():
	var user = username_input.text.strip_edges()
	var pass = password_input.text
	if user.length() < 3 or pass.length() < 6:
		status_label.text = "Username (3+) and password (6+) required"
		return
	_set_buttons_disabled(true)
	status_label.text = "Registering..."
	AuthManager.register(user, pass)

func _on_login_success():
	status_label.text = "Login successful!"
	WebSocketClient.connect_to_server()
	get_tree().change_scene_to_file("res://scenes/lobby/lobby.tscn")

func _on_login_failed(error: String):
	status_label.text = error
	_set_buttons_disabled(false)

func _on_register_success():
	status_label.text = "Registered! Connecting..."
	WebSocketClient.connect_to_server()
	get_tree().change_scene_to_file("res://scenes/lobby/lobby.tscn")

func _on_register_failed(error: String):
	status_label.text = error
	_set_buttons_disabled(false)

func _set_buttons_disabled(disabled: bool):
	login_button.disabled = disabled
	register_button.disabled = disabled
