extends Control

@onready var username_label: Label = %UsernameLabel
@onready var gold_label: Label = %GoldLabel
@onready var room_list: VBoxContainer = %RoomList
@onready var status_label: Label = %StatusLabel

var rooms: Array = []
var current_filter: int = 0

# Material You palette
const COLOR_SURFACE2 := Color(0.071, 0.149, 0.267, 1)
const COLOR_SURFACE3 := Color(0.094, 0.188, 0.333, 1)
const COLOR_BORDER   := Color(0.2, 0.32, 0.48, 1)
const COLOR_GOLD     := Color(0.831, 0.686, 0.216, 1)
const COLOR_TEXT     := Color(0.91, 0.88, 0.82, 1)
const COLOR_MUTED    := Color(0.55, 0.62, 0.72, 1)
const COLOR_GREEN    := Color(0.196, 0.537, 0.282, 1)
const COLOR_RED      := Color(0.773, 0.306, 0.306, 1)
const COLOR_TEAL     := Color(0.196, 0.686, 0.647, 1)

func _ready():
username_label.text = AuthManager.username
gold_label.text = "✦  %s" % _fmt_gold(10000)
# Update avatar letter
var avatar = get_node_or_null(
"Layout/Header/HeaderMargin/HeaderRow/AvatarCircle/AvatarLetter")
if avatar and AuthManager.username.length() > 0:
avatar.text = AuthManager.username[0].to_upper()
WebSocketClient.message_received.connect(_on_ws_message)
WebSocketClient.connected.connect(_on_ws_connected)
WebSocketClient.disconnected.connect(_on_ws_disconnected)
_fetch_profile()
if WebSocketClient._connected:
_on_ws_connected()

func _fmt_gold(g: int) -> String:
var s = str(g)
var result = ""
var count = 0
for i in range(s.length() - 1, -1, -1):
if count > 0 and count % 3 == 0:
result = "," + result
result = s[i] + result
count += 1
return result + " gold"

func _fetch_profile():
var url = NetworkConfig.base_url + "/api/user/profile"
var http = HTTPRequest.new()
add_child(http)
http.request_completed.connect(_on_profile_loaded.bind(http))
http.request(url, AuthManager.get_auth_headers(), HTTPClient.METHOD_GET)

func _on_profile_loaded(_result: int, response_code: int,
_headers: PackedStringArray, body: PackedByteArray, http: HTTPRequest):
http.queue_free()
if response_code == 200:
var json = JSON.new()
json.parse(body.get_string_from_utf8())
var data = json.get_data()
var g = data.get("gold_balance", 0)
gold_label.text = "✦  %s" % _fmt_gold(g)

func _on_ws_connected():
status_label.text = "✓ Connected"
status_label.modulate = Color(0.3, 0.8, 0.4, 1)
_fetch_rooms()

func _on_ws_disconnected():
status_label.text = "⚠ Reconnecting..."
status_label.modulate = Color(0.9, 0.6, 0.2, 1)

func _fetch_rooms():
var url = NetworkConfig.base_url + "/api/rooms?all=true"
if current_filter > 0:
url += "&ante=%d" % current_filter
var http = HTTPRequest.new()
add_child(http)
http.request_completed.connect(_on_rooms_loaded.bind(http))
http.request(url, AuthManager.get_auth_headers(), HTTPClient.METHOD_GET)

func _on_rooms_loaded(_result: int, response_code: int,
_headers: PackedStringArray, body: PackedByteArray, http: HTTPRequest):
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

if rooms.is_empty():
var empty_lbl = Label.new()
empty_lbl.text = "No rooms available"
empty_lbl.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
empty_lbl.modulate = COLOR_MUTED
empty_lbl.custom_minimum_size = Vector2(0, 60)
room_list.add_child(empty_lbl)
return

for room in rooms:
var card = _create_room_card(room)
room_list.add_child(card)

func _create_room_card(room: Dictionary) -> Control:
var margin = MarginContainer.new()
margin.theme_override_constants_margin_left = 0
margin.add_theme_constant_override("margin_left", 16)
margin.add_theme_constant_override("margin_right", 16)
margin.add_theme_constant_override("margin_top", 4)
margin.add_theme_constant_override("margin_bottom", 4)

var panel = PanelContainer.new()
var sbox = StyleBoxFlat.new()
sbox.bg_color = COLOR_SURFACE2
sbox.corner_radius_top_left = 16
sbox.corner_radius_top_right = 16
sbox.corner_radius_bottom_left = 16
sbox.corner_radius_bottom_right = 16
sbox.border_width_left = 1
sbox.border_width_top = 1
sbox.border_width_right = 1
sbox.border_width_bottom = 1
sbox.border_color = COLOR_BORDER
sbox.shadow_color = Color(0, 0, 0, 0.25)
sbox.shadow_size = 4
sbox.shadow_offset = Vector2(0, 2)
panel.add_theme_stylebox_override("panel", sbox)
panel.custom_minimum_size = Vector2(0, 72)
margin.add_child(panel)

var inner = MarginContainer.new()
inner.add_theme_constant_override("margin_left", 16)
inner.add_theme_constant_override("margin_right", 16)
inner.add_theme_constant_override("margin_top", 0)
inner.add_theme_constant_override("margin_bottom", 0)
panel.add_child(inner)

var row = HBoxContainer.new()
row.add_theme_constant_override("separation", 12)
inner.add_child(row)

# Ante dot
var ante_lbl = Label.new()
var ante = room.get("ante_amount", 0)
ante_lbl.text = "%dG" % ante
ante_lbl.custom_minimum_size = Vector2(52, 0)
ante_lbl.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
ante_lbl.add_theme_font_size_override("font_size", 13)
match ante:
100:  ante_lbl.modulate = COLOR_GOLD
500:  ante_lbl.modulate = COLOR_RED
1000: ante_lbl.modulate = COLOR_TEAL
_:    ante_lbl.modulate = COLOR_MUTED
row.add_child(ante_lbl)

# Room info column
var info = VBoxContainer.new()
info.size_flags_horizontal = Control.SIZE_EXPAND_FILL
info.add_theme_constant_override("separation", 2)
row.add_child(info)

var name_lbl = Label.new()
name_lbl.text = room.get("name", "Room")
name_lbl.modulate = COLOR_TEXT
name_lbl.add_theme_font_size_override("font_size", 15)
info.add_child(name_lbl)

var sub_row = HBoxContainer.new()
sub_row.add_theme_constant_override("separation", 10)
info.add_child(sub_row)

var count = room.get("player_count", 0)
var pcount_lbl = Label.new()
pcount_lbl.text = "👤 %d/4" % count
pcount_lbl.modulate = COLOR_MUTED
pcount_lbl.add_theme_font_size_override("font_size", 12)
sub_row.add_child(pcount_lbl)

if room.get("has_bots", false):
var bot_lbl = Label.new()
bot_lbl.text = "🤖 BOT"
bot_lbl.modulate = Color(0.6, 0.4, 0.85, 1)
bot_lbl.add_theme_font_size_override("font_size", 12)
sub_row.add_child(bot_lbl)

# Phase badge
var phase = room.get("phase", "LOBBY")
var phase_lbl = Label.new()
phase_lbl.text = phase
phase_lbl.add_theme_font_size_override("font_size", 11)
match phase:
"LOBBY":   phase_lbl.modulate = COLOR_GREEN
"PLAYING": phase_lbl.modulate = COLOR_RED
_:         phase_lbl.modulate = COLOR_MUTED
row.add_child(phase_lbl)

# Join button
var btn = Button.new()
btn.text = "JOIN"
btn.custom_minimum_size = Vector2(70, 40)
var btn_style = StyleBoxFlat.new()
btn_style.bg_color = COLOR_GOLD
btn_style.corner_radius_top_left = 20
btn_style.corner_radius_top_right = 20
btn_style.corner_radius_bottom_left = 20
btn_style.corner_radius_bottom_right = 20
btn.add_theme_stylebox_override("normal", btn_style)
btn.add_theme_stylebox_override("hover", btn_style)
btn.add_theme_stylebox_override("pressed", btn_style)
btn.add_theme_color_override("font_color", Color(0.08, 0.08, 0.08, 1))
btn.add_theme_font_size_override("font_size", 13)
var room_id = room.get("id", 0)
btn.pressed.connect(_on_room_join.bind(room_id))
row.add_child(btn)

# Animate card in
margin.modulate.a = 0.0
var tween = create_tween()
tween.tween_property(margin, "modulate:a", 1.0, 0.18)
return margin

func _on_room_join(room_id: int):
WebSocketClient.join_room(room_id)
status_label.text = "Joining room..."
status_label.modulate = COLOR_TEAL

func _on_ws_message(msg: Dictionary):
var type = msg.get("type", "")
match type:
"match_found":
status_label.text = "✓ Match found! Loading..."
status_label.modulate = COLOR_GREEN
get_tree().change_scene_to_file("res://scenes/game_room/game_room.tscn")
"room_update":
_fetch_rooms()
"card_dealt", "game_state":
get_tree().change_scene_to_file("res://scenes/game_room/game_room.tscn")
"error":
var payload = msg.get("payload", {})
status_label.text = "✗ " + payload.get("error", "Unknown error")
status_label.modulate = COLOR_RED

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
status_label.text = "🔍 Finding match (100G)..."
status_label.modulate = COLOR_GOLD

func _on_auto_match_500():
WebSocketClient.auto_match(500)
status_label.text = "🔍 Finding match (500G)..."
status_label.modulate = COLOR_RED

func _on_auto_match_1000():
WebSocketClient.auto_match(1000)
status_label.text = "🔍 Finding match (1000G)..."
status_label.modulate = COLOR_TEAL
