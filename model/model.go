package model

/* Application: Authentication
 * file: model
 */

/* --- Model Node Trees ---
 * User
 */

import (
	"context"
	"io/ioutil"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"

	jwt "github.com/dgrijalva/jwt-go"

	"golang.org/x/crypto/bcrypt"

	"google.golang.org/api/iterator"
)

/* collection<users>/doc<user.id>
 * Id use email
 * Pass hased and salt
 */
type User struct {
	Id     string    `firestore:"-" json:"-"`
	Name   string    `firestore:"name" json:"name"`
	Email  string    `firestore:"email" json:"email"`
	Pass   string    `firestore:"pass" json:"pass"`
	Role   int       `firestore:"role" json:"role,string"`
	Date   time.Time `firestore:"date" json:"date"`
	Update time.Time `firestore:"update,serverTimestamp" json:"update"`
}

func (u *User) Add(ctx context.Context, client *firestore.Client) error {
	pwd := u.Pass
	hashedPwd, err := hashAndSalt(pwd)
	if err != nil {
		return err
	}
	u.Pass = hashedPwd
	u.Name = strings.Split(u.Email, "@")[0]
	u.Date = time.Now()
	_, err = client.Collection("users").Doc(u.Id).Set(ctx, u)
	if err != nil {
		return err
	}
	return nil
}
func (u *User) Delete(ctx context.Context, client *firestore.Client) error {
	_, err := client.Collection("users").Doc(u.Id).Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}

/* Fetch
 * offset: From the specified number to the end
 */
func (u *User) Fetch(ctx context.Context, client *firestore.Client, offset int) (rep []*User, err error) {
	users := client.Collection("users")
	query := users.OrderBy("update", firestore.Desc).Offset(offset)
	iter := query.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		usr := &User{}
		if err = doc.DataTo(usr); err != nil {
			return nil, err
		}
		rep = append(rep, usr)
	}
	return rep, nil
}

/* --- password encryption --- */
func hashAndSalt(pwd string) (string, error) {
	// MinCost (4) DefaultCost(10) MaxCost(14)
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
func comparePasswords(hashedPwd string, pwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(pwd))
}

/* --- fs_query_defiition --- */
func GetEntity(ctx context.Context, client *firestore.Client, collection, docId string) (val map[string]interface{}, err error) {
	dsnap, err := client.Collection(collection).Doc(docId).Get(ctx)
	if err != nil {
		return nil, err
	}
	val = dsnap.Data()
	return val, nil
}

/* --- Storage handling --- */
func readStorage(ctx context.Context, bucketName, object string) ([]byte, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	rc, err := client.Bucket(bucketName).Object(object).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return ioutil.ReadAll(rc)
}

/* --- Jwt handling ---
 * j.Object shortens the storage object key.
 */
type Jwt struct {
	BucketName string
	Object     string
	Uid        string
	Role       int
	Expat      time.Time
	Issuer     string
	Token      string
}

type CustomClaim struct {
	Email string `json:"email"`
	Role  int    `json:"role"`
	*jwt.StandardClaims
}

/* Create a new token object, specifying signing method and the claims */
func (j *Jwt) Create(ctx context.Context) (string, error) {
	pkey, err := readStorage(ctx, j.BucketName, j.Object)
	if err != nil {
		return "", err
	}

	signature, err := jwt.ParseRSAPrivateKeyFromPEM(pkey)
	if err != nil {
		return "", err
	}
	// Create a new token object, specifying signing method and the claims
	claims := &CustomClaim{
		j.Uid,
		j.Role,
		&jwt.StandardClaims{
			ExpiresAt: j.Expat.Unix(),
			Issuer:    j.Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	tokenString, err := token.SignedString(signature)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (j *Jwt) Decode(ctx context.Context) error {
	pubkey, err := readStorage(ctx, j.BucketName, j.Object)
	if err != nil {
		return err
	}

	verifyKey, err := jwt.ParseRSAPublicKeyFromPEM(pubkey)
	if err != nil {
		return err
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(j.Token, &CustomClaim{}, func(token *jwt.Token) (interface{}, error) {
		return verifyKey, nil
	})
	if err != nil {
		return err
	}
	claims := token.Claims.(*CustomClaim)
	j.Uid = claims.Email
	j.Role = claims.Role

	return nil
}
