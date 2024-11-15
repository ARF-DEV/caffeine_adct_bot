package audio

var AirHornDefault OpusSound

func init() {
	var err error
	AirHornDefault, err = LoadSound("airhorn.dca")
	if err != nil {
		panic(err)
	}
}
