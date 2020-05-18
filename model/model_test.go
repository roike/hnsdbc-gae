package model

/* Application: Authentication
 * file: model_test
 * go test or go test -v
 */

import (
	"context"
	"encoding/json"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	projectID     = "transportsdn"
	defaultBucket = "transportsdn.appspot.com"
)

func TestUser(t *testing.T) {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	must := func(f func(context.Context, *firestore.Client) error) {
		err := f(ctx, client)
		if err != nil {
			fn := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
			t.Fatalf("must is %s: %v", fn, err)
		}
	}

	//--- Test entity ---
	t.Logf("Add User.")
	u := &User{}
	u.Email = "test.xa@example.com"
	u.Id = u.Email
	u.Pass = "123455678b"
	u.Role = 5
	u.Name = strings.Split(u.Email, "@")[0]
	u.Date = time.Now()
	must(u.Add)

	t.Logf("Get User.")
	user, err := GetEntity(ctx, client, "users", "test.xa@example.com")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("CompareHashPassword")
	if err = comparePasswords(user["pass"].(string), "123455678b"); err != nil {
		t.Errorf(" Failed compareHashPassword. %#v; ", err)
	}
	t.Logf("Fetch Users.")
	users, err := u.Fetch(ctx, client, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) == 0 {
		t.Errorf("Failed to fetch users.")
	}
	// orderby update desc
	b, err := json.Marshal(users[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("users: %s", b)

	t.Logf("Delete User.")
	err = u.Delete(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
}

func TestJwt(t *testing.T) {
	ctx := context.Background()
	j := Jwt{}
	j.BucketName = defaultBucket
	j.Object = path.Join("signature", "id_rsa")
	j.Uid = "test.a@example.com"
	j.Role = 5
	j.Expat = time.Now().Add(time.Second * 5)
	j.Issuer = "test"

	/* --- Create token --- */
	t.Logf("Create JWT token with PrivateKey.")
	tokenString, err := j.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}

	/* --- Decode Jwt token --- */
	// Override time value for test, set j.Expat 0 second.
	t.Logf("Decode JWT token with PublicKey.")
	j = Jwt{}
	j.BucketName = defaultBucket
	j.Object = path.Join("signature", "id_rsa.pub.pkcs8")
	j.Token = tokenString

	err = j.Decode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if j.Uid != "test.a@example.com" {
		t.Errorf("Uid is %#v; want %#v", j.Uid, "test.a@example.com")
	}

}

func TestSignedUrl(t *testing.T) {
	ctx := context.Background()
	v4 := SignedUrlV4{}
	v4.DefaultBucket = defaultBucket
	v4.UplodableBucket = "uploadable"
	v4.ObjectKey = path.Join("signature", "gcs-manager.json")
	v4.ContentType = "Content-Type:text/csv"
	v4.FileNameExt = "test.csv"
	v4.Expat = time.Now().Add(15 * time.Minute)
	err := v4.GenV4SignedUrl(ctx)
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf(v4.SignedUrl)
}
