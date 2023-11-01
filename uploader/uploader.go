package uploader

/*
import (

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/joho/godotenv"

)

//  It's important to note that all of the files must be updated/uploaded TOGETHER!
//  This is because the parser links all of the data together with ObjectID references, and
//  these references will change and cause things to break if files are updated/uploaded individually!

//  Also note that this uploader assumes that the collection names match the names of these files, which they should.
//  If the names of these collections ever change, the file names should be updated accordingly.

var filesToUpload []string = []string{"courses.json", "professors.json", "sections.json"}
*/
func Upload(inDir string, replace bool) {
	/*
		//Load env vars
		if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
		}
		//Connect to mongo
		client := connectDB()
		for _, path := range(filesToUpload) {
		//Open data file for reading
		fptr, err := os.Open(path)
		if err != nil {
		panic(err)
		}
		defer fptr.Close()
		}
	*/
}
