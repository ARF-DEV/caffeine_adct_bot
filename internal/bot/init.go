package bot

import "github.com/ARF-DEV/caffeine_adct_bot/internal/musicplayer"

func init() {
	var err error
	airHornDefault, err = musicplayer.LoadSound("airhorn.dca")
	if err != nil {
		panic(err)
	}
}
