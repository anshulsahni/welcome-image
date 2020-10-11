package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"

	"github.com/nfnt/resize"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"

	"github.com/gin-gonic/gin"
)

type GithubResponse struct {
	Name       string
	Avatar_url string
}

type InstagramResponse struct {
	Graphql struct {
		User struct {
			Full_name          string
			Profile_pic_url_hd string
		}
	}
}

const fontSize = 16
const smallFontSize = 7

func main() {
	server := gin.Default()
	server.GET("/ping", func(context *gin.Context) {

		provider, userID := getProviderAndUserID(context)

		name, displayPicture := getNameAndDpFromSocialMedia(provider, userID)
		fmt.Println(name, "Name of person")
		dpInReader, contentType := fetchDisplayPicture(displayPicture)

		sampleImage := superImposeDpOnRandomBackground(decodeDpImage(dpInReader, contentType))

		// welcomeMsg :=+ "Hi, " + name + " here, and I do what I do..."
		// addLabelToImage(sampleImage, name)
		fillRelevantTextInImage(sampleImage, name)

		buffer := new(bytes.Buffer)

		jpeg.Encode(buffer, sampleImage, nil)

		context.DataFromReader(http.StatusOK, int64(len(buffer.Bytes())), "image/jpeg", buffer, map[string]string{})
	})

	server.Run(":" + os.Getenv("PORT"))
}

func fetchDisplayPicture(dpLink string) (imageReader io.Reader, contentType string) {
	response, fetchingErr := http.Get(dpLink)
	fmt.Println(dpLink)
	if fetchingErr != nil {
		fmt.Println("Error in fetching display picture", fetchingErr)
		panic(1)
	}

	defer response.Body.Close()
	responseBytes, parsingErr := ioutil.ReadAll(response.Body)

	if parsingErr != nil {
		fmt.Println("Error in parsing image from response", parsingErr)
	}

	imageReader = bytes.NewReader(responseBytes)

	contentType = http.DetectContentType(responseBytes)

	return
}

func decodeDpImage(dpReader io.Reader, contentType string) image.Image {
	var dpImage image.Image
	var decodingErr error
	switch contentType {
	case "image/png":
		dpImage, decodingErr = png.Decode(dpReader)

	case "image/jpeg":
		dpImage, decodingErr = jpeg.Decode(dpReader)
	}

	if decodingErr != nil {
		fmt.Println("Error in decoding image", decodingErr)
		panic(1)
	}

	return dpImage
}

func superImposeDpOnRandomBackground(dpImage image.Image) *image.RGBA {
	backgroundImg := createRandomBg(1000, 400)

	dpSuperImposeArea := image.Rect(400, 50, 600, 250)
	dpImage = resize.Resize(200, 200, dpImage, resize.Lanczos2)

	draw.Draw(backgroundImg, dpSuperImposeArea, dpImage, dpImage.Bounds().Min, draw.Over)

	return backgroundImg
}

func createRandomBg(width int, height int) *image.RGBA {
	red, blue, green := getRandomColorCombo()
	background := color.RGBA{red, blue, green, 255}

	rect := image.Rect(0, 0, width, height)
	img := image.NewRGBA(rect)

	draw.Draw(img, img.Bounds(), &image.Uniform{background}, image.ZP, draw.Src)
	return img
}

func getRandomColorCombo() (red, blue, green uint8) {
	red = uint8(rand.Intn(255))
	blue = uint8(rand.Intn(255))
	green = uint8(rand.Intn(255))
	return
}

func getProviderAndUserID(context *gin.Context) (string, string) {
	provider := context.DefaultQuery("provider", "facebook")
	userID := context.Query("user_id")

	return provider, userID
}

func getNameAndDpFromSocialMedia(provider, userID string) (string, string) {
	name, dpUrl := parseFetchedDetails(getUserDetails(provider, userID), provider)
	return name, dpUrl
}

func getUserDetails(provider, userID string) []byte {
	var providerUrl string
	switch provider {

	case "instagram":
		providerUrl = "https://instagram.com/" + userID + "/?__a=1"

	case "github":
		providerUrl = "https://api.github.com/users/" + userID
	}

	response, err := http.Get(providerUrl)

	if err != nil {
		fmt.Println("Error in fetching user information from provider", err)
		panic(1)
	}

	defer response.Body.Close()
	responseBytes, readingErr := ioutil.ReadAll(response.Body)

	if readingErr != nil {
		fmt.Println("Error in reading from response", readingErr)
		panic(1)
	}

	return responseBytes
}

func parseFetchedDetails(responseBytes []byte, provider string) (string, string) {
	var parsingErr error
	var name, dpLink string

	switch provider {
	case "github":
		var data GithubResponse
		parsingErr = json.Unmarshal(responseBytes, &data)
		name = data.Name
		dpLink = data.Avatar_url

	case "instagram":
		var data InstagramResponse
		parsingErr = json.Unmarshal(responseBytes, &data)
		name = data.Graphql.User.Full_name
		dpLink = data.Graphql.User.Profile_pic_url_hd
	}

	if parsingErr != nil {
		fmt.Println("Error in parsing json", parsingErr)
		panic(1)
	}

	return name, dpLink
}

func fillRelevantTextInImage(img *image.RGBA, name string) {
	xForName := (1000 - (fontSize * len(name))) / 2
	addLabelToImage(img, name, "./user-name.ttf", xForName, 300, false)

	label := "Software Developer @ Razorpay"

	xForJobDesig := (1000 - (fontSize * len(label))) / 2
	addLabelToImage(img, label, "./other-content.ttf", xForJobDesig, 350, false)

	addLabelToImage(img, "Fun fact: Everytime you refresh you'll get a new color in background", "./post-script.ttf", 10, 390, true)
}

func addLabelToImage(img *image.RGBA, label string, fontFile string, x, y int, small bool) {
	fontSizeToBeUsed := float64(fontSize)
	if small {
		fontSizeToBeUsed = float64(smallFontSize)
	}

	freetypeContext := freetype.NewContext()

	point := freetype.Pt(x, y)

	freetypeContext.SetDst(img)
	freetypeContext.SetSrc(image.NewUniform(color.Black))
	freetypeContext.SetDPI(144.00)
	freetypeContext.SetClip(img.Bounds())

	freetypeContext.SetFont(getParseFont(getFontBytes(fontFile)))

	freetypeContext.SetFontSize(float64(fontSizeToBeUsed))

	if _, err := freetypeContext.DrawString(label, point); err != nil {
		fmt.Println("Error in drawing text over image", err)
		panic(1)
	}

}

func getParseFont(fontBytes []byte) *truetype.Font {
	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		fmt.Println("Error during parsing font", err)
		panic(1)
	}

	return font
}

func getFontBytes(fontFile string) []byte {
	fontBytes, err := ioutil.ReadFile(fontFile)
	if err != nil {
		fmt.Println("Error during reading font file", err)
		panic(1)
	}

	return fontBytes
}

func getContrastWithWhite(clr color.Color) {
	// wrtie here
}

func getGammChannel(colorInInt float64) float64 {
	if colorInInt <= 10 {
		return colorInInt / 3294
	}
	return math.Pow(((colorInInt / 269) + 0.0513), 2.4)
}
