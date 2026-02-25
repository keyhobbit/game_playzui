extends Control

@onready var username_input: LineEdit = %UsernameInput
@onready var password_input: LineEdit = %PasswordInput
@onready var login_button: Button = %LoginButton
@onready var register_button: Button = %RegisterButton
@onready var status_label: Label = %StatusLabel

const COLOR_GOLD  := Color(0.831, 0.686, 0.216, 1)
const COLOR_TEAL  := Color(0.306, 0.8, 0.769, 1)
const COLOR_ERROR := Color(0.878, 0.35, 0.35, 1)


func _ready():
AuthManager.login_success.connect(_on_login_success)
AuthManager.login_failed.connect(_on_login_failed)
AuthManager.register_success.connect(_on_register_success)
AuthManager.register_failed.connect(_on_register_failed)

# Entrance animation: card slides up + fades in
var card = get_node_or_null("Center/Card")
if card:
var start_y = card.position.y + 40
card.position.y = start_y
card.modulate.a = 0.0
var tw = create_tween().set_parallel(true)
tw.tween_property(card, "modulate:a", 1.0, 0.35)
tw.tween_property(card, "position:y", card.position.y - 40, 0.35) \
.set_trans(Tween.TRANS_BACK).set_ease(Tween.EASE_OUT)

# Animate decorative suit symbols
_animate_deco("DecoSpade",   0.0)
_animate_deco("DecoHeart",   0.1)
_animate_deco("DecoClub",    0.2)
_animate_deco("DecoDiamond", 0.3)


func _animate_deco(node_name: String, delay: float):
var node = get_node_or_null(node_name)
if node:
node.modulate.a = 0.0
var tw = create_tween()
tw.tween_property(node, "modulate:a", node.modulate.a + 0.07, 0.5).set_delay(delay + 0.4)


func _on_login_pressed():
var user = username_input.text.strip_edges()
var pass = password_input.text
if user.length() < 3 or pass.length() < 6:
_show_status("Username (3+) and password (6+) required", COLOR_ERROR)
_shake_card()
return
_set_buttons_disabled(true)
_show_status("Logging in...", COLOR_TEAL)
AuthManager.login(user, pass)


func _on_register_pressed():
var user = username_input.text.strip_edges()
var pass = password_input.text
if user.length() < 3 or pass.length() < 6:
_show_status("Username (3+) and password (6+) required", COLOR_ERROR)
_shake_card()
return
_set_buttons_disabled(true)
_show_status("Creating account...", COLOR_TEAL)
AuthManager.register(user, pass)


func _on_login_success():
_show_status("✓ Login successful!", COLOR_GOLD)
WebSocketClient.connect_to_server()
get_tree().change_scene_to_file("res://scenes/lobby/lobby.tscn")


func _on_login_failed(error: String):
_show_status("✗ " + error, COLOR_ERROR)
_shake_card()
_set_buttons_disabled(false)


func _on_register_success():
_show_status("✓ Account created! Connecting...", COLOR_GOLD)
WebSocketClient.connect_to_server()
get_tree().change_scene_to_file("res://scenes/lobby/lobby.tscn")


func _on_register_failed(error: String):
_show_status("✗ " + error, COLOR_ERROR)
_shake_card()
_set_buttons_disabled(false)


func _show_status(text: String, color: Color):
status_label.text = text
status_label.modulate = color


func _shake_card():
var card = get_node_or_null("Center/Card")
if not card:
return
var origin = card.position.x
var tw = create_tween()
tw.tween_property(card, "position:x", origin + 10, 0.05)
tw.tween_property(card, "position:x", origin - 10, 0.05)
tw.tween_property(card, "position:x", origin + 6,  0.04)
tw.tween_property(card, "position:x", origin - 6,  0.04)
tw.tween_property(card, "position:x", origin,      0.04)


func _set_buttons_disabled(disabled: bool):
login_button.disabled = disabled
register_button.disabled = disabled
