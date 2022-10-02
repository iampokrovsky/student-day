package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Loc struct {
	msgs     map[string]string
	aliases  map[string]string
	objects  map[string]map[string]func(loc *Loc) string
	things   [][]string
	quests   []string
	avlLocs  []string
	blkdLocs []string
}

func (loc *Loc) getThingsList() string {
	var list []string
	for _, storage := range loc.things {
		if len(storage) > 1 {
			list = append(list, storage[0]+" "+strings.Join(storage[1:], ", "))
		}
	}
	if len(list) == 0 {
		return loc.msgs["empty"]
	}
	return strings.Join(list, ", ")
}

func (loc *Loc) getQuestsList() string {
	var str string
	lgth := len(loc.quests)

	if lgth > 0 {
		str += loc.quests[lgth-1]
	}

	if lgth > 1 {
		str = strings.Join(loc.quests[:lgth-1], ", ") + " и " + str
	}

	if str != "" {
		str = "надо " + str
	}

	return str
}

func (loc *Loc) getAvlLocs() string {
	locs := make([]string, 0, len(loc.avlLocs))
	for _, l := range loc.avlLocs {
		if alias, aliasExists := loc.aliases[l]; aliasExists {
			l = alias
		}
		locs = append(locs, l)
	}
	return "можно пройти - " + strings.Join(locs, ", ")
}

func (loc *Loc) getInfoStr(list ...string) string {
	var sb strings.Builder
	var sep string

	for i, str := range list {
		if str != "" {
			sb.WriteString(sep)
			sb.WriteString(str)
			sep = ", "
		}

		if i == len(list)-2 {
			sep = ". "
		}
	}

	return sb.String()
}

type User struct {
	curLoc string
	hasBp  bool
	things []string
}

type Game struct {
	user   User
	locs   map[string]*Loc
	quests map[string]func(game *Game) bool
}

func (game *Game) checkQuests(loc *Loc) {
	var rest []string
	for _, quest := range loc.quests {
		cb, cbExists := game.quests[quest]
		if !cbExists || !cb(game) {
			rest = append(rest, quest)
		}
	}
	loc.quests = rest
}

func (game *Game) lookAround() string {
	loc := game.locs[game.user.curLoc]
	game.checkQuests(loc)
	return loc.getInfoStr(loc.msgs["lookAround"], loc.getThingsList(), loc.getQuestsList(), loc.getAvlLocs())
}

func (game *Game) move(locName string) string {
	loc, locExists := game.locs[locName]
	if !locExists {
		return "такой комнаты не существует"
	}

	curLoc := game.locs[game.user.curLoc]

	var isAvlLoc bool
	for _, avlLoc := range curLoc.avlLocs {
		if avlLoc == locName {
			isAvlLoc = true
		}
	}

	var isBlkd bool
	for _, blkdLoc := range curLoc.blkdLocs {
		if blkdLoc == locName {
			isBlkd = true
		}
	}

	if !isAvlLoc {
		return fmt.Sprintf("нет пути в %s", locName)
	}

	if isAvlLoc && isBlkd {
		return "дверь закрыта"
	}

	game.user.curLoc = locName
	return loc.getInfoStr(loc.msgs["move"], loc.getAvlLocs())
}

func (game *Game) getThingFromLoc(loc Loc, thing string) bool {
	for i, place := range loc.things {
		for j, item := range place {
			if j != 0 && item == thing {
				loc.things[i] = append(loc.things[i][:j], loc.things[i][j+1:]...)
				return true
			}
		}
	}
	return false
}

func (game *Game) take(thing string) string {
	if !game.user.hasBp {
		return "некуда класть"
	}
	thingExists := game.getThingFromLoc(*game.locs[game.user.curLoc], thing)
	if thingExists {
		game.user.things = append(game.user.things, thing)
		return fmt.Sprintf("предмет добавлен в инвентарь: %s", thing)
	}
	return "нет такого"
}

func (game *Game) putOnBp() string {
	bpExists := game.getThingFromLoc(*game.locs[game.user.curLoc], "рюкзак")
	if bpExists {
		game.user.things = append(game.user.things, "рюкзак")
		game.user.hasBp = true
		return "вы надели: рюкзак"
	}
	return "нет такого"
}

func (game *Game) apply(thing, object string) string {
	var thingExists bool
	for _, userThing := range game.user.things {
		if userThing == thing {
			thingExists = true
		}
	}

	loc := game.locs[game.user.curLoc]
	obj, objExists := loc.objects[object]
	cb, cbExists := obj[thing]

	switch {
	case !thingExists:
		return fmt.Sprintf("нет предмета в инвентаре - %s", thing)
	case !objExists:
		return "не к чему применить"
	case cbExists:
		return cb(loc)
	default:
		return "нет действия"
	}
}

func (game *Game) handleCommand(input string) string {
	in := strings.Fields(input)
	cmd, params := in[0], in[1:]
	paramsAmt := len(params)
	res := "неизвестная команда"

	switch {
	case cmd == "осмотреться":
		switch {
		case paramsAmt > 0:
			res = "лишние параметры"
		default:
			res = game.lookAround()
		}
	case cmd == "идти":
		switch {
		case paramsAmt < 1:
			res = "не указана комната"
		case paramsAmt > 1:
			res = "должен быть только 1 параметр"
		default:
			res = game.move(params[0])
		}
	case cmd == "взять":
		switch {
		case paramsAmt < 1:
			res = "не указана предмет"
		case paramsAmt > 1:
			res = "должен быть только 1 параметр"
		default:
			res = game.take(params[0])
		}
	case cmd == "надеть" && params[0] == "рюкзак":
		switch {
		case paramsAmt > 1:
			res = "лишние параметры"
		default:
			res = game.putOnBp()
		}
	case cmd == "применить":
		switch {
		case paramsAmt < 2:
			res = "не хватает параметров"
		case paramsAmt > 2:
			res = "должен быть только 2 параметра"
		default:
			res = game.apply(params[0], params[1])
		}
	}
	return res
}

func (game *Game) listenInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		game.handleCommand(scanner.Text())
	}
}

func initGame() {
	game = Game{
		user: User{
			curLoc: "кухня",
		},
		locs: map[string]*Loc{
			"кухня": &Loc{
				msgs: map[string]string{
					"move":       "кухня, ничего интересного",
					"lookAround": "ты находишься на кухне",
				},
				things:  [][]string{{"на столе:", "чай"}},
				quests:  []string{"собрать рюкзак", "идти в универ"},
				avlLocs: []string{"коридор"},
			},
			"коридор": &Loc{
				msgs: map[string]string{
					"move": "ничего интересного",
				},
				objects: map[string]map[string]func(loc *Loc) string{
					"дверь": {
						"ключи": func(loc *Loc) string {
							for i, blkd := range loc.blkdLocs {
								if blkd == "улица" {
									loc.blkdLocs = append(loc.blkdLocs[:i], loc.blkdLocs[i+1:]...)
								}
							}
							loc.avlLocs = append(loc.avlLocs, "улица")
							return "дверь открыта"
						},
					},
				},
				avlLocs:  []string{"кухня", "комната", "улица"},
				blkdLocs: []string{"улица"},
			},
			"комната": &Loc{
				msgs: map[string]string{
					"move":  "ты в своей комнате",
					"empty": "пустая комната",
				},
				things:  [][]string{{"на столе:", "ключи", "конспекты"}, {"на стуле:", "рюкзак"}},
				avlLocs: []string{"коридор"},
			},
			"улица": &Loc{
				msgs: map[string]string{
					"move": "на улице весна",
				},
				aliases: map[string]string{
					"коридор": "домой",
				},
				avlLocs: []string{"коридор"},
			},
		},
		quests: map[string]func(game *Game) bool{
			"собрать рюкзак": func(game *Game) bool {
				things := []string{"рюкзак", "конспекты"}
				var isDone bool
				for _, thing := range game.user.things {
					if thing == things[0] || thing == things[1] {
						isDone = true
					}
				}
				return isDone
			},
		},
	}
}

func handleCommand(input string) string {
	return game.handleCommand(input)
}

var game Game

func main() {
	initGame()
	game.listenInput()
}
