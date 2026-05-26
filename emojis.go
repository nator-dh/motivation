package main

import "math/rand/v2"

var motivationalEmojis = []string{
	"💪", "🔥", "🚀", "⚡", "✨", "🌟", "⭐", "🏆", "🎯", "🧠",
	"💡", "🦁", "🦅", "🌅", "🌄", "🏔️", "⛰️", "🧗", "🏋️", "🥇",
	"📈", "🛠️", "⚒️", "🗻", "🌊", "🌱", "🍀", "💎", "🔑", "🧭",
	"⏳", "🎖️", "🏅", "🌞", "☀️", "🦾", "🤝", "🙌", "👊", "✊",
}

// RandomEmoji returns a single motivational emoji at random.
func RandomEmoji() string {
	return motivationalEmojis[rand.IntN(len(motivationalEmojis))]
}
