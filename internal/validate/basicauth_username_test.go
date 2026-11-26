package validate

import "testing"

func TestBasicAuthUsernameRejectsDelimiters(t *testing.T){
	for _,name:=range []string{"bad:name","bad\nname","bad\rname","bad\x00name"}{
		if BasicAuthUsername(name)==nil{t.Fatalf("accepted %q",name)}
	}
	if err:=BasicAuthUsername("admin");err!=nil{t.Fatal(err)}
}
