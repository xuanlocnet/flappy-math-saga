package main

import (
	"encoding/json"
	"github.com/gopherjs/gopherjs/js"
	jQueryStatic "github.com/rusco/jquery"
	"math"
	"strconv"
	"strings"
)

type Object map[string]interface{}

var (
	jQuery = jQueryStatic.NewJQuery
)

const (
	STATESSPLASHSCREEN = 0
	STATESGAMESSCREEN  = 1
	STATESSCORESCREEN  = 2
)

var (
	//state
	currentstate = 0
	//params
	gravity         = 0.3 //was:0.25
	velocity        = 0.0
	position        = 180.0
	rotation        = 0
	jump            = -4.6
	score           = 0
	highscore       = 0
	pipeHeight      = 400 //was: 300
	pipewidth       = 52
	pipes           = []jQueryStatic.JQuery{}
	replayclickable = false
	//sounds
	volume      = 30
	soundJump   = js.Global.Get("buzz").Get("sound").New("assets/sounds/sfx_wing.ogg")
	soundScore  = js.Global.Get("buzz").Get("sound").New("assets/sounds/sfx_point.ogg")
	soundHit    = js.Global.Get("buzz").Get("sound").New("assets/sounds/sfx_hit.ogg")
	soundDie    = js.Global.Get("buzz").Get("sound").New("assets/sounds/sfx_die.ogg")
	soundSwoosh = js.Global.Get("buzz").Get("sound").New("assets/sounds/sfx_swooshing.ogg")
	//eventIds
	loopGameloop float64
	loopPipeloop float64
)

func main() {

	jQuery().Ready(func() {

		if js.Global.Get("location").Get("search").String() == "?easy" {
			pipeHeight = 200
		}

		//get the highscore
		var savedscore int
		getStore("highscore", &savedscore)
		if savedscore != 0 {
			highscore = savedscore
		}

		//start with the splash screen
		showSplash()

		onKeyDown()
		replayOnClick()
	})
}

func setStore(key string, val interface{}) {
	byteArr, _ := json.Marshal(val)
	str := string(byteArr)
	js.Global.Get("localStorage").Call("setItem", key, str)
}
func getStore(key string, val interface{}) {
	item := js.Global.Get("localStorage").Call("getItem", key)
	if item.IsNull() {
		val = nil
		return
	}
	str := item.String()
	json.Unmarshal([]byte(str), &val)
}

func showSplash() {

	currentstate = STATESSPLASHSCREEN

	//set the defaults (again)
	velocity = 0
	position = 180.0
	rotation = 0
	score = 0

	//update the player in preparation for the next game
	jQuery("#player").SetCss(map[string]interface{}{"x": 0, "y": 0})
	updatePlayer(jQuery("#player"))

	soundSwoosh.Call("stop")
	soundSwoosh.Call("play")

	//clear out all the pipes if there are any
	jQuery(".pipe").Remove()
	pipes = make([]jQueryStatic.JQuery, 0) //new Array()

	//make everything animated again
	jQuery(".animated").SetCss("animation-play-state", "running")
	jQuery(".animated").SetCss("-webkit-animation-play-state", "running")

	//fade in the splash
	jQuery("#splash").Underlying().Call("transition", Object{"opacity": "1"}, 2000, "ease")
}

func startGame() {
	currentstate = STATESGAMESSCREEN

	//fade out the splash
	jQuery("#splash").Stop()

	jQuery("#splash").Underlying().Call("transition", Object{"opacity": "0"}, 500, "ease")

	//update the big score
	setBigScore(false)

	//start up our loops
	updaterate := 1000.0 / 60.0 //60 times/sec

	loopGameloop = js.Global.Call("setInterval", gameloop, updaterate).Float()
	loopPipeloop = js.Global.Call("setInterval", updatePipes, 2600).Float()

	//jump from the start!
	playerJump()
	updatePipes()
}

func updatePlayer(player jQueryStatic.JQuery) {
	//rotation
	rotation = int(math.Min(float64((velocity/10)*90), 90))

	//apply rotation and position
	player.SetCss(map[string]interface{}{"rotate": rotation, "top": position})
}

func gameloop() {
	var player = jQuery("#player")

	//update the player speed/position
	velocity += gravity
	position += velocity

	//update the player
	updatePlayer(player)

	//create the bounding boxtop
	var box = js.Global.Get("document").Call("getElementById", "player").Call("getBoundingClientRect")
	var origwidth = 34.0
	var origHeight = 24.0

	var boxwidth = origwidth - (math.Sin(math.Abs(float64(rotation))/90.0) * 8.0)

	var boxheight = (origHeight + box.Get("height").Float()) / 2.0
	var boxleft = ((box.Get("width").Float() - boxwidth) / 2.0) + box.Get("left").Float()

	var boxtop = ((box.Get("height").Float() - boxheight) / 2.0) + box.Get("top").Float()
	var boxright = boxleft + boxwidth

	var boxbottom = boxtop + boxheight

	if box.Get("bottom").Float() >= float64(jQuery("#land").Offset().Top) {
		playerDead()
		return
	}
	//have they tried to escape through the ceiling? :o
	var ceiling = jQuery("#ceiling")
	if boxtop <= (float64(ceiling.Offset().Top) + float64(ceiling.Height())) {
		position = 0.0
	}
	//we can"t go any further without a pipe
	if len(pipes) == 0 {
		return
	}
	//determine the bounding box of the next pipes inner area
	var nextpipe = pipes[0]

	var nextpipeupper = nextpipe.Children(".pipe_upper")
	var nextpipemiddle = nextpipe.Children(".pipe_middle")

	var middletop = float64(nextpipemiddle.Offset().Top)
	var middlebottom = middletop + float64(nextpipemiddle.Height())

	var pipetop = float64(nextpipeupper.Offset().Top) + float64(nextpipeupper.Height())

	var pipeleft = nextpipeupper.Offset().Left - 2.0 // for some reason it starts at the inner pipes offset, not the outer pipes
	var piperight = float64(pipeleft) + float64(pipewidth)
	var pipebottom = float64(pipetop) + float64(pipeHeight)

	//have we gotten inside the pipe yet?

	if boxright > float64(pipeleft) {
		//we"re within the pipe, which pipes have we passed through?

		if boxtop > float64(pipetop) && boxbottom < float64(middletop) {
			//we"re in the top gap
			if nextpipe.Data("correct").(float64) == 1.0 {
				//top guess is correct
				//pass through
			} else {

				playerDead()
				return
			}

		} else if boxbottom < float64(pipebottom) && boxtop > float64(middlebottom) {
			//we"re in the bottom gap
			if nextpipe.Data("correct").(float64) == 0.0 {
				//bottom guess is correct
				//pass through
			} else {

				playerDead()
				return
			}
		} else {
			//no! we touched the pipe
			playerDead()
			return
		}
	}

	//have we passed the imminent danger?
	if boxleft > float64(piperight) {
		//yes, remove it
		//pipes.splice(0, 1)
		pipes = pipes[1:len(pipes)]
		//and score a point
		playerScore()
	}

}

//Handle space bar
func onKeyDown() {

	jQuery(js.Global.Get("document")).On(jQueryStatic.KEYDOWN, func(e jQueryStatic.Event) {
		//space bar!

		if e.KeyCode == 32 {
			//in ScoreScreen, hitting space should click the "replay" button. else it"s just a regular spacebar hit
			if currentstate == STATESSCORESCREEN {
				jQuery("#replay").Trigger(jQueryStatic.CLICK)
			} else {
				screenClick()
			}
		}
	})

	//Handle mouse down OR touch start
	//2do: test on touch device !
	if !js.Global.Get("ontouchstart").IsUndefined() {
		jQuery(js.Global.Get("document")).On(jQueryStatic.TOUCHSTART, screenClick)
	} else {
		jQuery(js.Global.Get("document")).On(jQueryStatic.MOUSEDOWN, screenClick)
	}

}

func screenClick() {
	if currentstate == STATESGAMESSCREEN {
		playerJump()
	} else if currentstate == STATESSPLASHSCREEN {
		startGame()
	}
}

func playerJump() {
	velocity = jump
	//play jump sound
	soundJump.Call("stop")
	soundJump.Call("play")
}

func setBigScore(erase bool) {

	elemscore := jQuery("#bigscore")
	elemscore.Empty()

	if erase {
		return
	}

	scoreStr := strconv.Itoa(score)
	digits := strings.Split(scoreStr, "")
	for i := 0; i < len(digits); i++ {
		elemscore.Append("<img src='assets/font_big_" + digits[i] + ".png' alt='" + digits[i] + "'>")
	}

}

func setSmallScore() {

	elemscore := jQuery("#currentscore")
	elemscore.Empty()

	scoreStr := strconv.Itoa(score)
	digits := strings.Split(scoreStr, "")

	for i := 0; i < len(digits); i++ {
		elemscore.Append("<img src='assets/font_small_" + digits[i] + ".png' alt='" + digits[i] + "'>")
	}

}

func setHighScore() {

	elemscore := jQuery("#highscore")
	elemscore.Empty()

	scoreStr := strconv.Itoa(score)
	digits := strings.Split(scoreStr, "")
	for i := 0; i < len(digits); i++ {
		elemscore.Append("<img src='assets/font_small_" + digits[i] + ".png' alt='" + digits[i] + "'>")
	}

}

func setMedal() bool {

	elemmedal := jQuery("#medal")
	elemmedal.Empty()

	if score < 10 {
		//signal that no medal has been won
		return false
	}

	var medal string

	if score >= 10 {
		medal = "bronze"
	}
	if score >= 20 {
		medal = "silver"
	}
	if score >= 30 {
		medal = "gold"
	} else {
		medal = "platinum"
	}
	elemmedal.Append("<img src='assets/medal_" + medal + ".png' alt='" + medal + "'>")

	return true
}

func playerDead() {

	//stop animating everything!
	jQuery(".animated").SetCss("animation-play-state", "paused")
	jQuery(".animated").SetCss("-webkit-animation-play-state", "paused")

	//drop the bird to the floor
	playerbottom := float64(jQuery("#player").Position().Top) + float64(jQuery("#player").Width()) //we use width because he"ll be rotated 90 deg
	floor := jQuery("#flyarea").Height()

	movey := math.Max(0.0, float64(float64(floor)-playerbottom))
	moveyStr := strconv.FormatFloat(movey, 'g', 1, 64)
	jQuery("#player").Underlying().Call("transition", Object{"y": moveyStr + "px", "rotate": "90"}, 1000, "easeInOutCubic")

	//it"s time to change states_ as of now we"re considered ScoreScreen to disable left click/flying
	currentstate = STATESSCORESCREEN

	//destroy our gameloops
	js.Global.Call("clearInterval", loopGameloop)
	js.Global.Call("clearInterval", loopPipeloop)
	loopGameloop = 0
	loopPipeloop = 0

	soundHit.Call("play")
	soundHit.Call("stop")
	showScore()

}

func showScore() {

	jQuery("#scoreboard").SetCss("display", "block")

	//remove the big score
	setBigScore(true)

	//have they beaten their high score?
	if score > highscore {
		//yeah!
		highscore = score
		//save it!
		setStore("highscore", highscore)
	}

	//update the scoreboard
	setSmallScore()
	setHighScore()
	wonmedal := setMedal()

	//SWOOSH!
	soundSwoosh.Call("stop")
	soundSwoosh.Call("play")

	//show the scoreboard
	//move it down so we can slide it up
	jQuery("#scoreboard").SetCss(Object{"y": "40px", "opacity": "0"})
	jQuery("#replay").SetCss(Object{"y": "40px", "opacity": "0"})

	jQuery("#scoreboard").Underlying().Call("transition", Object{"y": "0px", "opacity": "1"}, 600, "ease", func() {
		//When the animation is done, animate in the replay button and SWOOSH!
		soundSwoosh.Call("stop")
		soundSwoosh.Call("play")

		jQuery("#replay").Underlying().Call("transition", Object{"y": "0px", "opacity": "1"}, 600, "ease")

		//also animate in the MEDAL! WOO!
		if wonmedal {
			jQuery("#medal").SetCss(map[string]interface{}{"scale": "2", "opacity": "0"})
			jQuery("#medal").Underlying().Call("transition", Object{"scale": "1", "opacity": "1"}, 1200, "ease")
		}
	})

	//make the replay button clickable
	replayclickable = true

}

func replayOnClick() {
	jQuery("#replay").On(jQueryStatic.CLICK, func() {
		//make sure we can only click once
		if !replayclickable {
			return
		} else {
			replayclickable = false
		}
		//SWOOSH!
		soundSwoosh.Call("stop")
		soundSwoosh.Call("play")

		//fade out the scoreboard
		jQuery("#scoreboard").Underlying().Call("transition", Object{"y": "-40px", "opacity": "0"}, 1000, "ease", func() {
			//when that"s done, display us back to nothing
			jQuery("#scoreboard").SetCss("display", "none")

			//start the game over!
			showSplash()
		})
	})

}
func playerScore() {
	score += 1
	//play score sound
	soundScore.Call("stop")
	soundScore.Call("play")
	setBigScore(false)
}

func randomIntFromInterval(min, max int) int {

	rnd := js.Global.Get("Math").Call("random").Float()
	maxplusmin := float64(max - min + 1)
	rnd = rnd*maxplusmin + float64(min)
	return int(math.Floor(rnd))
}

func random0to1() float64 {
	return js.Global.Get("Math").Call("random").Float()
}

func updatePipes() {

	//Do any pipes need removal?
	jQuery(".pipe").Filter(func() bool {
		return jQuery(js.This).Position().Left <= -100
	}).Remove()

	//add a new pipe (top Height + bottom Height  + pipeHeight == 420) and put it in our tracker
	var padding = 20
	var constraint = 420 - pipeHeight - (padding * 2)                                   //double padding (for top and bottom)
	var topHeight = math.Floor((random0to1() * float64(constraint)) + float64(padding)) //add lower padding
	var bottomHeight = (420 - int(pipeHeight)) - int(topHeight)
	var middleHeight = pipeHeight / 3.0
	var middletop = int(topHeight) + int(pipeHeight/3.0)

	//pipe skeleton

	bh := strconv.Itoa(bottomHeight + 35)
	th := strconv.Itoa(int(topHeight) + 35)
	thStr := strconv.Itoa(int(topHeight))
	bhStr := strconv.Itoa(bottomHeight)
	mhStr := strconv.Itoa(middleHeight)
	mtStr := strconv.Itoa(middletop)

	var html = `<div class="pipe animated"><div class="pipe_upper" style="height: ` + thStr +
		`px;"></div><div class="guess top" style="top: ` + th + `px;"></div><div class="pipe_middle" style="height: ` + mhStr +
		`px; top: ` + mtStr +
		`px;"></div><div class="guess bottom" style="bottom: ` + bh + `px;"></div><div class="pipe_lower" style="height: ` + bhStr + `px;"></div><div class="question"></div></div>`
	var newpipe = jQuery(html)

	//generate two random numbers
	var firstnumber = randomIntFromInterval(2, 10)
	var secondnumber = randomIntFromInterval(2, 10)

	var firstnumber_digits = strings.Split(strconv.Itoa(firstnumber), "")
	var secondnumber_digits = strings.Split(strconv.Itoa(secondnumber), "")

	//append first number of question
	for i := 0; i < len(firstnumber_digits); i++ {

		htx0 := `<div class = "question_digit first" style = "background-image: url('assets/font_big_` + firstnumber_digits[i] + `.png');"></div>`
		newpipe.Children(".question").Append(htx0)
	}

	//append multiplication symbol
	newpipe.Children(".question").Append(`<div class="question_digit symbol" style="background-image: url('assets/font_shitty_x.png');"></div>`)

	//append second number of question
	for i := 0; i < len(secondnumber_digits); i++ {

		htx1 := `<div class = "question_digit second" style = "background-image: url('assets/font_big_` + secondnumber_digits[i] + `.png');"></div>`
		newpipe.Children(".question").Append(htx1)
	}

	//generate a correct and incorrect guess

	correctguess := firstnumber * secondnumber
	smaller := int(math.Min(float64(firstnumber), float64(secondnumber)))
	offset := smaller * randomIntFromInterval(1, 4)

	incorrectguess := 0

	if randomIntFromInterval(0, 1) == 1 {
		incorrectguess = correctguess + int(offset)
	} else {
		incorrectguess = correctguess - int(offset)
		if incorrectguess <= 0 {
			incorrectguess = correctguess + int(offset)
		}
	}

	//flip a coin - 1: top is correct, 0: bottom is correct
	topguesscorrect := randomIntFromInterval(0, 1)

	//append first guess
	correctguess_digits := strings.Split(strconv.Itoa(correctguess), "")
	incorrectguess_digits := strings.Split(strconv.Itoa(incorrectguess), "")

	for i := 0; i < len(correctguess_digits); i++ {

		var newdigit = jQuery(`<div class = "guess_digit" style = "background-image: url('assets/font_big_` + correctguess_digits[i] + `.png');"></div>`)

		if topguesscorrect == 1 {
			newpipe.Children(".guess.top").Append(newdigit)
		} else {
			newpipe.Children(".guess.bottom").Append(newdigit)
		}
	}

	//append second guess
	for i := 0; i < len(incorrectguess_digits); i++ {

		var newdigit = jQuery(`<div class = "guess_digit" style = "background-image: url('assets/font_big_` + incorrectguess_digits[i] + `.png');"></div>`)
		if topguesscorrect == 1 {
			newpipe.Children(".guess.bottom").Append(newdigit)
		} else {
			newpipe.Children(".guess.top").Append(newdigit)
		}
	}

	jQuery("#flyarea").Append(newpipe)
	newpipe.SetData("correct", topguesscorrect)
	pipes = append(pipes, newpipe)
}
