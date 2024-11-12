package musicplayer

var airHornDefault OpusSound

func init() {
	var err error
	airHornDefault, err = LoadSound("airhorn.dca")
	if err != nil {
		panic(err)
	}
}
