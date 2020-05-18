package main

/* Application: Authentication Server
file: main
how to run:
go run .
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/crypto/bcrypt"

	md "hnsdbc/model"
	rt "github.com/roike/go-util/router"
)

var (
	projectID        string
	defaultBucket    string
	userApiKey       string
	userApiDecodeKey string
)

func main() {
	projectID = os.Getenv("PROJECT_ID")
	defaultBucket = os.Getenv("DEFAULT_BUCKET")
	userApiKey = path.Join("signature", "id_rsa")
	userApiDecodeKey = path.Join("signature", "id_rsa.pub.pkcs8")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))

}

func init() {
	r := rt.New("/")
	r.Handle("GET", "/error", errorHandle)
	r.Handle("POST", "/login", postAuth)
	r.Handle("POST", "/user", putUser)
	r.Handle("POST", "/user/repassword", putRepassword)
	r.Handle("GET", "/users/:offset", fetchUsers)
	r.Handle("POST", "/user/delete", deleteUser)

	r.PanicHandler = panicHandle
	r.Wrapper = checkAuthen

	http.Handle("/", r)
}

func panicHandle(w http.ResponseWriter, r *http.Request, p interface{}) {
	log.Printf("Raised panic %v", p)
}

/* Wrapper
 * Redirect to defaultHandler if there is no matching url.
 */
func checkAuthen(r *http.Request, urlPath string) (string, error) {
	// Skip
	if urlPath == "/login" {
		return urlPath, nil
	}
	// End Skip

	if strings.HasPrefix(urlPath, "/user") {
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		token := splitToken[1]
		if token == "" {
			log.Printf("Request is without token.")
			return "/error", nil
		}
		ctx := context.Background()
		j := md.Jwt{}
		j.BucketName = defaultBucket
		j.Object = userApiDecodeKey
		j.Token = token
		err := j.Decode(ctx)
		if err != nil {
			log.Printf("Request'token is invalidi. Error: %s", err)
			return "/error", nil
		}
		if j.Role == 5 {
			return urlPath, nil
		}
		if urlPath == "/user/repassword" {
			return urlPath, nil
		}
	}

	return "/", nil
}

var errorHandle rt.AppHandle = func(w io.Writer, r *http.Request, _ rt.Param) error {
	tmpl, err := template.ParseFiles(path.Join("static", "error.html"))
	if err != nil {
		log.Fatalf("Not found files : %v", err)
	}
	messageMap := map[string]string{
		"message": "This is off limits.",
	}
	return tmpl.Execute(w, messageMap)
}

/* POST:/user
 */
var putUser rt.AppHandle = func(w io.Writer, r *http.Request, _ rt.Param) error {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	u := &md.User{}
	err = json.NewDecoder(r.Body).Decode(u)
	if err != nil {
		log.Printf("Failed to put user: %v", err)
		return fmt.Errorf("Failed to put user: %v", err)
	}
	defer r.Body.Close()
	u.Id = u.Email
	err = u.Add(ctx, client)
	if err != nil {
		return fmt.Errorf("Failed to put user: %v", err)
	}

	return nil
}

/* POST:/repassword
 */
var putRepassword rt.AppHandle = func(w io.Writer, r *http.Request, _ rt.Param) error {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	var result interface{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&result); err != nil {
		log.Printf("Failed to get parameters: %v", err)
		return fmt.Errorf("Failed to get email&pass.")
	}
	defer r.Body.Close()
	payload, _ := result.(map[string]interface{})
	user, err := md.GetEntity(ctx, client, "users", payload["email"].(string))
	if err != nil {
		log.Printf("Failed to get entity: %v", err)
		return fmt.Errorf("Failed to get entity.")
	}
	hashedPwd := user["pass"].(string)
	pwd := payload["pass"].(string)
	err = bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(pwd))
	if err != nil {
		log.Printf("Failed to decode password: %v", err)
		return rt.AppErrorf(http.StatusUnauthorized, "Original pasword is wrong.")
	}
	u := &md.User{}
	u.Id = user["email"].(string)
	u.Email = user["email"].(string)
	u.Pass = payload["pass2"].(string)
	u.Role = int(user["role"].(int64))
	err = u.Add(ctx, client)
	if err != nil {
		log.Printf("Failed to save rePassword: %v", err)
		return fmt.Errorf("Failed to put user: %v", err)
	}

	return nil
}

/* GET:/user/delete
 */
var deleteUser rt.AppHandle = func(w io.Writer, r *http.Request, _ rt.Param) error {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	u := &md.User{}
	err = json.NewDecoder(r.Body).Decode(u)
	if err != nil {
		log.Printf("Failed to put user: %v", err)
		return fmt.Errorf("Failed to put user: %v", err)
	}
	defer r.Body.Close()
	u.Id = u.Email
	err = u.Delete(ctx, client)
	if err != nil {
		return fmt.Errorf("Failed to put user: %v", err)
	}

	return nil
}

/* GET:/users/:offset
 */
var fetchUsers rt.AppHandle = func(w io.Writer, r *http.Request, p rt.Param) error {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	offset := 0
	if poffset, ok := p["offset"]; ok {
		offset, _ = strconv.Atoi(poffset)
	}
	u := md.User{}
	users, err := u.Fetch(ctx, client, offset)
	if err != nil {
		return fmt.Errorf("Failed to fetch Users: %v", err)
	}

	return json.NewEncoder(w).Encode(users)
}

/* POST:/login
 * r.BODY: {"email":"test.a@example.com","pass":"12345678"}
 */
var postAuth rt.AppHandle = func(w io.Writer, r *http.Request, _ rt.Param) error {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	r.ParseForm()
	user, err := md.GetEntity(ctx, client, "users", r.Form.Get("email"))
	if err != nil {
		log.Printf("Failed to get entity: %v", err)
		return fmt.Errorf("Failed to get entity.")
	}
	hashedPwd := user["pass"].(string)
	pwd := r.Form.Get("password")
	err = bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(pwd))
	if err != nil {
		log.Printf("Failed to decode password given: %v", pwd)
		//log.Printf("Failed to decode password hash: %v", hashedPwd)
		log.Printf("Failed to decode password: %v", err)

		return rt.AppErrorf(http.StatusUnauthorized, "Pasword is wrong.")
	}
	jwt := &md.Jwt{}
	jwt.BucketName = defaultBucket
	jwt.Object = userApiKey
	jwt.Uid = user["email"].(string)
	jwt.Role = int(user["role"].(int64))
	jwt.Expat = time.Now().Add(time.Hour * 24)
	jwt.Issuer = "transportsdn"

	token, err := jwt.Create(ctx)
	if err != nil {
		return fmt.Errorf("failed: %v", err)
	}

	reply := map[string]string{
		"token": token,
		"email": user["email"].(string),
		"role":  strconv.Itoa(int(user["role"].(int64))),
	}
	return json.NewEncoder(w).Encode(reply)
}
