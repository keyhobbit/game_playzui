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

const CARD_WIDTH = 45
const CARD_HEIGHT = 65
const CARD_COLORS = {
	"H": Color(0.9, 0.15, 0.15),
	"D": Color(0.9, 0.15, 0.15),
	"C": Color(0.1, 0.1, 0.1),
	"S": Color(0.1, 0.1, 0.1)
}
const SUIT_SYMBOLS = {"H": "♥", "D": "♦", "C": "♣", "S": "♠"}

func _ready():
	WebSocketClient.message_received.connect(_on_ws_message)
	_update_ui()

func _process(delta: float):
	if is_my_turn and game_phase == "PLAYING":
		turn_time_left -= delta
		turn_timer_label.text = str(max(0, int(turn_time_left)))
		if turn_time_left <= 0:
			is_my_turn = false

func _on_ws_message(msg: Dictionary):
	var type = msg.get("type", "")
	var payload = msg.get("payload", {})

	match type:
		"card_dealt":
			_handle_card_dealt(payload)
		"game_state":
			_handle_game_state(payload)
		"room_update":
			_handle_room_update(payload)
		"move_played":
			_handle_move_played(payload)
		"turn_change":
			_handle_turn_change(payload)
		"settlement":
			_handle_settlement(payload)
		"chat_relay":
			_handle_chat(payload)
		"error":
			turn_indicator.text = payload.get("error", "Error")

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
	var table_clear = payload.get("table_clear", false)
	if table_clear:
		_clear_table_animated()
	_update_ui()

func _handle_settlement(payload: Dictionary):
	game_phase = "SETTLEMENT"
	var winner = payload.get("winner", -1)
	var results = payload.get("results", [])
	var text = "Game Over! "
	for r in results:
		if r == null:
			continue
		var name = r.get("username", "?")
		var delta = r.get("gold_delta", 0)
		var prefix = "+" if delta > 0 else ""
		text += "%s: %s%d  " % [name, prefix, delta]
	turn_indicator.text = text
	_update_ui()

func _handle_chat(payload: Dictionary):
	var sender = payload.get("sender", "?")
	var message = payload.get("message", "")
	chat_log.append_text("[b]%s:[/b] %s\n" % [sender, message])

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
		var card_node = _create_card_node(card, true)
		hand_container.add_child(card_node)

func _render_hand_animated():
	for child in hand_container.get_children():
		child.queue_free()
	selected_cards.clear()

	for i in range(hand.size()):
		var card = hand[i]
		var card_node = _create_card_node(card, true)
		card_node.modulate.a = 0.0
		hand_container.add_child(card_node)

		var tween = create_tween()
		tween.tween_property(card_node, "modulate:a", 1.0, 0.15).set_delay(i * 0.05)

func _create_card_node(card: Dictionary, interactive: bool) -> Control:
	var panel = Panel.new()
	panel.custom_minimum_size = Vector2(CARD_WIDTH, CARD_HEIGHT)

	var style = StyleBoxFlat.new()
	style.bg_color = Color(0.95, 0.93, 0.88)
	style.border_width_left = 1
	style.border_width_right = 1
	style.border_width_top = 1
	style.border_width_bottom = 1
	style.border_color = Color(0.3, 0.3, 0.3)
	style.corner_radius_top_left = 4
	style.corner_radius_top_right = 4
	style.corner_radius_bottom_left = 4
	style.corner_radius_bottom_right = 4
	panel.add_theme_stylebox_override("panel", style)

	var suit_str = card.get("suit", "S")
	var rank_str = card.get("rank", "?")
	var color = CARD_COLORS.get(suit_str, Color.BLACK)
	var symbol = SUIT_SYMBOLS.get(suit_str, "?")

	var label = Label.new()
	label.text = rank_str + "\n" + symbol
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	label.add_theme_color_override("font_color", color)
	label.anchors_preset = Control.PRESET_FULL_RECT
	panel.add_child(label)

	if interactive and game_phase == "PLAYING":
		panel.mouse_filter = Control.MOUSE_FILTER_STOP
		panel.gui_input.connect(_on_card_clicked.bind(card, panel))

	panel.set_meta("card_data", card)
	return panel

func _on_card_clicked(event: InputEvent, card: Dictionary, panel: Panel):
	if not event is InputEventMouseButton:
		return
	if not event.pressed:
		return

	var idx = _find_card_in_selected(card)
	if idx >= 0:
		selected_cards.remove_at(idx)
		var tween = create_tween()
		tween.tween_property(panel, "position:y", 0, 0.1)
	else:
		selected_cards.append(card)
		var tween = create_tween()
		tween.tween_property(panel, "position:y", -20, 0.1)

func _find_card_in_selected(card: Dictionary) -> int:
	for i in range(selected_cards.size()):
		if selected_cards[i].get("rank", "") == card.get("rank", "") and \
		   selected_cards[i].get("suit", "") == card.get("suit", ""):
			return i
	return -1

func _animate_cards_to_table(cards: Array):
	for child in table_cards.get_children():
		child.queue_free()

	for card in cards:
		var card_node = _create_card_node(card, false)
		card_node.modulate.a = 0.0
		card_node.scale = Vector2(0.5, 0.5)
		table_cards.add_child(card_node)
		var tween = create_tween().set_parallel(true)
		tween.tween_property(card_node, "modulate:a", 1.0, 0.2)
		tween.tween_property(card_node, "scale", Vector2(1.0, 1.0), 0.2)

func _clear_table_animated():
	for child in table_cards.get_children():
		var tween = create_tween()
		tween.tween_property(child, "modulate:a", 0.0, 0.3)
		tween.tween_callback(child.queue_free)

func _update_players():
	var positions = [bottom_player, left_player, top_player, right_player]
	var seat_map = _get_relative_seats()

	for i in range(4):
		var pos = positions[i]
		var name_label = pos.get_child(0) as Label
		var count_label = pos.get_child(1) as Label
		var seat = seat_map[i]
		var player = _get_player_at_seat(seat)
		if player:
			name_label.text = player.get("username", "Empty")
			var cc = player.get("card_count", 0)
			if game_phase == "LOBBY":
				var ready = player.get("is_ready", false)
				count_label.text = "READY" if ready else "Not Ready"
			else:
				count_label.text = "Cards: %d" % cc
		else:
			name_label.text = "Empty"
			count_label.text = ""

func _get_relative_seats() -> Array:
	if my_seat < 0:
		return [0, 1, 2, 3]
	var seats = []
	for i in range(4):
		seats.append((my_seat + i) % 4)
	return seats

func _get_player_at_seat(seat: int):
	for p in players:
		if p.get("seat_index", -1) == seat:
			return p
	return null

func _update_ui():
	phase_label.text = game_phase
	is_my_turn = (current_turn == my_seat and game_phase == "PLAYING")

	ready_button.visible = (game_phase == "LOBBY" and my_seat >= 0)
	play_button.visible = is_my_turn
	pass_button.visible = is_my_turn

	if is_my_turn:
		turn_time_left = 30.0
		turn_indicator.text = "YOUR TURN"
	elif game_phase == "PLAYING":
		var p = _get_player_at_seat(current_turn)
		if p:
			turn_indicator.text = "%s's turn" % p.get("username", "?")
		else:
			turn_indicator.text = ""
	elif game_phase == "LOBBY":
		turn_indicator.text = "Waiting for players..."
		turn_timer_label.text = ""

func _on_ready_pressed():
	WebSocketClient.set_ready()

func _on_play_pressed():
	if selected_cards.size() == 0:
		turn_indicator.text = "Select cards to play"
		return
	WebSocketClient.play_cards(selected_cards)
	selected_cards.clear()

func _on_pass_pressed():
	WebSocketClient.pass_turn()

func _on_leave_pressed():
	WebSocketClient.leave_room()
	get_tree().change_scene_to_file("res://scenes/lobby/lobby.tscn")

func _on_send_chat():
	var msg = chat_input.text.strip_edges()
	if msg.length() > 0:
		WebSocketClient.send_chat(msg)
		chat_input.text = ""

func _on_chat_toggle():
	chat_panel.visible = !chat_panel.visible
