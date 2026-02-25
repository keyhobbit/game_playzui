extends Control

@onready var table_cards: HBoxContainer = %TableCards
@onready var hand_container: HBoxContainer = %HandContainer
@onready var top_player: VBoxContainer = %TopPlayer
@onready var left_player: VBoxContainer = %LeftPlayer
@onready var right_player: VBoxContainer = %RightPlayer
@onready var bottom_player: VBoxContainer = %BottomPlayer
@onready var ready_button: Button = %ReadyButton
@onready var play_button: Button = %PlayButton
@onready var pass_button: Button = %PassButton
@onready var turn_indicator: Label = %TurnIndicator
@onready var turn_timer_label: Label = %TurnTimer
@onready var phase_label: Label = %PhaseLabel
@onready var chat_panel: VBoxContainer = %ChatPanel
@onready var chat_log: RichTextLabel = %ChatLog
@onready var chat_input: LineEdit = %ChatInput

var hand: Array = []
var selected_cards: Array = []
var game_phase: String = "LOBBY"
var current_turn: int = -1
var my_seat: int = -1
var players: Array = []
var turn_time_left: float = 30.0
var is_my_turn: bool = false

# Card dimensions for Kenney medium assets (scaled)
const CARD_W := 64
const CARD_H := 88

# Kenney suits
const SUIT_MAP := {"S": "spades", "C": "clubs", "D": "diamonds", "H": "hearts"}
const RANK_MAP := {
"2": "02", "3": "03", "4": "04", "5": "05", "6": "06",
"7": "07", "8": "08", "9": "09", "10": "10",
"J": "J", "Q": "Q", "K": "K", "A": "A"
}
const CARD_ASSET_PATH := "res://assets/cards/PNG/Cards (medium)/"

# Fallback text rendering colors
const CARD_COLORS := {
"H": Color(0.85, 0.12, 0.12), "D": Color(0.85, 0.12, 0.12),
"C": Color(0.1,  0.1,  0.1),  "S": Color(0.1,  0.1,  0.1)
}
const SUIT_SYMBOLS := {"H": "♥", "D": "♦", "C": "♣", "S": "♠"}


func _ready():
WebSocketClient.message_received.connect(_on_ws_message)
_update_ui()


func _process(delta: float):
if is_my_turn and game_phase == "PLAYING":
turn_time_left -= delta
var secs = max(0, int(turn_time_left))
turn_timer_label.text = str(secs)
if secs <= 10:
turn_timer_label.modulate = Color(0.9, 0.3, 0.3, 1)
else:
turn_timer_label.modulate = Color(0.91, 0.88, 0.82, 1)
if turn_time_left <= 0:
is_my_turn = false


func _on_ws_message(msg: Dictionary):
var type = msg.get("type", "")
var payload = msg.get("payload", {})
match type:
"card_dealt":     _handle_card_dealt(payload)
"game_state":     _handle_game_state(payload)
"room_update":    _handle_room_update(payload)
"move_played":    _handle_move_played(payload)
"turn_change":    _handle_turn_change(payload)
"settlement":     _handle_settlement(payload)
"chat_relay":     _handle_chat(payload)
"error":
turn_indicator.text = "✗ " + payload.get("error", "Error")
turn_indicator.modulate = Color(0.9, 0.35, 0.35, 1)


func _handle_card_dealt(payload: Dictionary):
hand = payload.get("hand", [])
players = payload.get("players", [])
game_phase = payload.get("phase", "PLAYING")
current_turn = payload.get("current_turn", -1)
_find_my_seat()
_render_hand_animated()
_update_players()
_update_ui()


func _handle_game_state(payload: Dictionary):
game_phase = payload.get("phase", "LOBBY")
current_turn = payload.get("current_turn", -1)
players = payload.get("players", [])
var h = payload.get("hand", [])
if h.size() > 0:
hand = h
_find_my_seat()
_render_hand()
_update_players()
_update_ui()


func _handle_room_update(payload: Dictionary):
if payload.has("players"):
players = payload.get("players", [])
game_phase = payload.get("phase", "LOBBY")
_find_my_seat()
_update_players()
_update_ui()


func _handle_move_played(payload: Dictionary):
var cards = payload.get("cards", [])
_animate_cards_to_table(cards)


func _handle_turn_change(payload: Dictionary):
current_turn = payload.get("current_turn", current_turn)
turn_time_left = 30.0
if payload.get("table_clear", false):
_clear_table_animated()
_update_ui()


func _handle_settlement(payload: Dictionary):
game_phase = "SETTLEMENT"
var results = payload.get("results", [])
var lines: Array = ["🏆 Game Over!"]
for r in results:
if r == null:
continue
var nm = r.get("username", "?")
var delta = r.get("gold_delta", 0)
var prefix = "+" if delta > 0 else ""
var penalty = r.get("penalty_multiplier", 1)
var line = "%s: %s%dG" % [nm, prefix, delta]
if penalty > 1:
line += "  💀 ×%d" % penalty
lines.append(line)
turn_indicator.text = "\n".join(lines)
turn_indicator.modulate = Color(0.831, 0.686, 0.216, 1)
_update_ui()


func _handle_chat(payload: Dictionary):
var sender = payload.get("sender", "?")
var message = payload.get("message", "")
chat_log.append_text("[color=#d4af37][b]%s:[/b][/color] %s\n" % [sender, message])


func _find_my_seat():
my_seat = -1
for p in players:
if p.get("user_id", 0) == AuthManager.user_id:
my_seat = p.get("seat_index", -1)
break


func _render_hand():
for child in hand_container.get_children():
child.queue_free()
selected_cards.clear()
for card in hand:
hand_container.add_child(_create_card_node(card, true))


func _render_hand_animated():
for child in hand_container.get_children():
child.queue_free()
selected_cards.clear()
for i in range(hand.size()):
var node = _create_card_node(hand[i], true)
node.modulate.a = 0.0
node.scale = Vector2(0.7, 0.7)
hand_container.add_child(node)
var tw = create_tween().set_parallel(true)
tw.tween_property(node, "modulate:a", 1.0, 0.18).set_delay(i * 0.04)
tw.tween_property(node, "scale", Vector2(1.0, 1.0), 0.18).set_delay(i * 0.04)


func _get_card_texture(card: Dictionary) -> Texture2D:
var suit_key = card.get("suit", "S")
var rank_key = card.get("rank", "3")
var suit_name = SUIT_MAP.get(suit_key, "spades")
var rank_name = RANK_MAP.get(rank_key, "03")
var path = "%scard_%s_%s.png" % [CARD_ASSET_PATH, suit_name, rank_name]
if ResourceLoader.exists(path):
return load(path) as Texture2D
return null


func _create_card_node(card: Dictionary, interactive: bool) -> Control:
var texture = _get_card_texture(card)
if texture:
# Use Kenney asset as TextureRect
var wrapper = PanelContainer.new()
wrapper.custom_minimum_size = Vector2(CARD_W, CARD_H)
var sbox = StyleBoxFlat.new()
sbox.bg_color = Color(0.95, 0.93, 0.88, 1)
sbox.corner_radius_top_left = 6
sbox.corner_radius_top_right = 6
sbox.corner_radius_bottom_left = 6
sbox.corner_radius_bottom_right = 6
sbox.shadow_color = Color(0, 0, 0, 0.45)
sbox.shadow_size = 5
sbox.shadow_offset = Vector2(0, 3)
wrapper.add_theme_stylebox_override("panel", sbox)

var tex = TextureRect.new()
tex.texture = texture
tex.stretch_mode = TextureRect.STRETCH_SCALE
tex.anchors_preset = Control.PRESET_FULL_RECT
tex.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
wrapper.add_child(tex)

if interactive and game_phase == "PLAYING":
wrapper.mouse_filter = Control.MOUSE_FILTER_STOP
wrapper.gui_input.connect(_on_card_clicked.bind(card, wrapper))
wrapper.set_meta("card_data", card)
return wrapper
else:
# Fallback: text card
var panel = PanelContainer.new()
panel.custom_minimum_size = Vector2(CARD_W, CARD_H)
var sbox = StyleBoxFlat.new()
sbox.bg_color = Color(0.95, 0.93, 0.88, 1)
sbox.border_width_left = 1
sbox.border_width_top = 1
sbox.border_width_right = 1
sbox.border_width_bottom = 1
sbox.border_color = Color(0.3, 0.3, 0.3, 1)
sbox.corner_radius_top_left = 6
sbox.corner_radius_top_right = 6
sbox.corner_radius_bottom_left = 6
sbox.corner_radius_bottom_right = 6
sbox.shadow_color = Color(0, 0, 0, 0.4)
sbox.shadow_size = 5
sbox.shadow_offset = Vector2(0, 3)
panel.add_theme_stylebox_override("panel", sbox)

var suit_str = card.get("suit", "S")
var rank_str = card.get("rank", "?")
var color = CARD_COLORS.get(suit_str, Color.BLACK)
var symbol = SUIT_SYMBOLS.get(suit_str, "?")

var lbl = Label.new()
lbl.text = "%s\n%s" % [rank_str, symbol]
lbl.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
lbl.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
lbl.add_theme_color_override("font_color", color)
lbl.add_theme_font_size_override("font_size", 15)
lbl.anchors_preset = Control.PRESET_FULL_RECT
lbl.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
panel.add_child(lbl)

if interactive and game_phase == "PLAYING":
panel.mouse_filter = Control.MOUSE_FILTER_STOP
panel.gui_input.connect(_on_card_clicked.bind(card, panel))
panel.set_meta("card_data", card)
return panel


func _on_card_clicked(event: InputEvent, card: Dictionary, node: Control):
if not event is InputEventMouseButton or not event.pressed:
return
var idx = _find_card_in_selected(card)
if idx >= 0:
selected_cards.remove_at(idx)
var tw = create_tween()
tw.tween_property(node, "position:y", 0.0, 0.1)
else:
selected_cards.append(card)
var tw = create_tween()
tw.tween_property(node, "position:y", -18.0, 0.1)


func _find_card_in_selected(card: Dictionary) -> int:
for i in range(selected_cards.size()):
if selected_cards[i].get("rank") == card.get("rank") and \
   selected_cards[i].get("suit") == card.get("suit"):
return i
return -1


func _animate_cards_to_table(cards: Array):
for child in table_cards.get_children():
child.queue_free()
for card in cards:
var node = _create_card_node(card, false)
node.modulate.a = 0.0
node.scale = Vector2(0.5, 0.5)
table_cards.add_child(node)
var tw = create_tween().set_parallel(true)
tw.tween_property(node, "modulate:a", 1.0, 0.22)
tw.tween_property(node, "scale", Vector2(1.0, 1.0), 0.22)


func _clear_table_animated():
for child in table_cards.get_children():
var tw = create_tween()
tw.tween_property(child, "modulate:a", 0.0, 0.28)
tw.tween_callback(child.queue_free)


func _get_player_at_seat(seat: int) -> Dictionary:
for p in players:
if p.get("seat_index", -1) == seat:
return p
return {}


func _get_relative_seats() -> Array:
# Returns [my, left, opposite, right] seats for [bottom, left, top, right]
if my_seat < 0:
return [0, 1, 2, 3]
return [my_seat, (my_seat + 3) % 4, (my_seat + 2) % 4, (my_seat + 1) % 4]


func _get_player_name_label(slot: VBoxContainer) -> Label:
# Navigate: VBox → Panel → Inner → VBox → Name (index 0)
if slot.get_child_count() == 0:
return null
var child0 = slot.get_child(0)
if child0 is Label:
return child0   # BottomPlayer has direct Label
# For Top/Left/Right: Panel -> Margin -> VBox -> Name
if child0 is PanelContainer and child0.get_child_count() > 0:
var margin = child0.get_child(0)
if margin and margin.get_child_count() > 0:
var vbox = margin.get_child(0)
if vbox and vbox.get_child_count() > 0:
return vbox.get_child(0) as Label
return null


func _get_player_cards_label(slot: VBoxContainer) -> Label:
var child0 = slot.get_child(0)
if child0 is Label:
return null  # BottomPlayer shows "You", no card count there
if child0 is PanelContainer and child0.get_child_count() > 0:
var margin = child0.get_child(0)
if margin and margin.get_child_count() > 0:
var vbox = margin.get_child(0)
if vbox and vbox.get_child_count() > 1:
return vbox.get_child(1) as Label
return null


func _update_players():
var positions = [bottom_player, left_player, top_player, right_player]
var seat_map = _get_relative_seats()

for i in range(4):
var slot = positions[i]
var seat = seat_map[i]
var player = _get_player_at_seat(seat)

var name_lbl = _get_player_name_label(slot)
var cards_lbl = _get_player_cards_label(slot)

if player.is_empty():
if name_lbl: name_lbl.text = "---"
if cards_lbl: cards_lbl.text = "🂠 --"
else:
var username = player.get("username", "?")
var card_count = player.get("card_count", 13)
var is_active = player.get("seat_index", -1) == current_turn
var is_bot = player.get("is_bot", false)

if name_lbl:
name_lbl.text = ("🤖 " if is_bot else "") + username
if is_active:
name_lbl.modulate = Color(0.831, 0.686, 0.216, 1)
else:
name_lbl.modulate = Color(0.91, 0.88, 0.82, 1)
if cards_lbl:
cards_lbl.text = "🂠 %d" % card_count


func _update_ui():
phase_label.text = game_phase
is_my_turn = (my_seat == current_turn)

match game_phase:
"LOBBY":
ready_button.visible = true
play_button.visible = false
pass_button.visible = false
turn_indicator.text = "Waiting for players..."
turn_indicator.modulate = Color(0.55, 0.65, 0.75, 1)
"PLAYING":
ready_button.visible = false
play_button.visible = is_my_turn
pass_button.visible = is_my_turn
if is_my_turn:
turn_indicator.text = "🎴 YOUR TURN"
turn_indicator.modulate = Color(0.831, 0.686, 0.216, 1)
turn_time_left = 30.0
else:
var p = _get_player_at_seat(current_turn)
turn_indicator.text = "%s's turn" % p.get("username", "...")
turn_indicator.modulate = Color(0.55, 0.65, 0.75, 1)
"SETTLEMENT":
ready_button.visible = false
play_button.visible = false
pass_button.visible = false


func _on_ready_pressed():
WebSocketClient.send_message({"type": "ready", "payload": {}})
ready_button.disabled = true
ready_button.text = "✓ READY"


func _on_play_pressed():
if selected_cards.is_empty():
turn_indicator.text = "Select cards first!"
return
WebSocketClient.send_message({"type": "play_cards", "payload": {"cards": selected_cards}})
selected_cards.clear()
_render_hand()


func _on_pass_pressed():
WebSocketClient.send_message({"type": "pass_turn", "payload": {}})


func _on_chat_submitted(text: String):
if text.strip_edges().is_empty():
return
WebSocketClient.send_message({"type": "chat", "payload": {"message": text}})
chat_input.clear()
