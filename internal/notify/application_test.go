package notify

import(
 "context"
 "encoding/json"
 "io"
 "net/http"
 "strings"
 "testing"
 "time"
)
func TestApplicationMessages(t *testing.T){
 for _,provider:=range []WebhookProvider{WebhookWeComApp,WebhookFeishuApp}{
  t.Run(string(provider),func(t *testing.T){
   authCalls,sendCalls:=0,0
   client:=&http.Client{Transport:robotTransport(func(r *http.Request)(*http.Response,error){
    body:=""
    if r.Body!=nil{b,_:=io.ReadAll(r.Body);body=string(b)}
    response:=""
    if strings.Contains(r.URL.Path,"gettoken")||strings.Contains(r.URL.Path,"tenant_access_token"){
     authCalls++
     if provider==WebhookWeComApp{if r.URL.Query().Get("corpsecret")!="secret-test"{t.Fatal("missing corpsecret")};response=`{"errcode":0,"access_token":"test-token","expires_in":7200}`}else{if !strings.Contains(body,"secret-test"){t.Fatal("missing app secret")};response=`{"code":0,"tenant_access_token":"test-token","expire":7200}`}
    }else{
     sendCalls++
     if provider==WebhookWeComApp{if r.URL.Query().Get("access_token")!="test-token"||!strings.Contains(body,`"touser":"test-user"`){t.Fatal("invalid WeCom message")};response=`{"errcode":0}`}else{if r.Header.Get("Authorization")!="Bearer test-token"||r.URL.Query().Get("receive_id_type")!="open_id"||!strings.Contains(body,`"receive_id":"test-user"`){t.Fatal("invalid Feishu message")};response=`{"code":0}`}
    }
    return &http.Response{StatusCode:200,Body:io.NopCloser(strings.NewReader(response)),Header:make(http.Header)},nil
   })}
   cfg:=RuntimeConfig{Config:Config{Webhook:WebhookConfig{Provider:provider,AppID:"app-test",AgentID:"1000002",Recipient:"test-user",RecipientType:"open_id"}},WebhookSecret:"secret-test"}
   for i:=0;i<2;i++{if err:=SendWebhook(context.Background(),client,cfg,FormatAlert("test","测试","内容"));err!=nil{t.Fatal(err)}}
   if authCalls!=1||sendCalls!=2{t.Fatalf("token cache not used: %d/%d",authCalls,sendCalls)}
  })
 }
}
func TestRobotCredentialIsolation(t *testing.T){
 old:=WebhookConfig{Provider:WebhookFeishu,Signed:true}
 encoded,_:=json.Marshal(robotCredential{Primary:"https://open.feishu.cn/open-apis/bot/v2/hook/test",Signature:"sig-test"})
 raw:=robotCredentialPrefix+string(encoded)
 resolved,err:=resolveWebhookCredential(old,MaskedSecret,old,raw)
 if err!=nil||resolved!=raw{t.Fatalf("saved signature not preserved: %v",err)}
 resolved,err=resolveWebhookCredential(WebhookConfig{Provider:WebhookWeComApp,AppID:"other"},MaskedSecret,old,raw)
 if err!=nil||resolved!=""{t.Fatal("credential reused across providers")}
 a:=WebhookConfig{Provider:WebhookWeComApp,AppID:"corp1"}
 b:=WebhookConfig{Provider:WebhookWeComApp,AppID:"corp2"}
 if v,_:=resolveWebhookCredential(b,MaskedSecret,a,"secret");v!=""{t.Fatal("app secret reused across app IDs")}
 payload:=map[string]any{}
 signRobot(WebhookFeishu,"https://example.com","sig-test",payload,time.Unix(1700000000,0))
 if payload["timestamp"]!="1700000000"||payload["sign"]==""{t.Fatal("missing Feishu signature")}
 endpoint:=signRobot(WebhookDingTalk,"https://example.com?access_token=x","sig-test",nil,time.Unix(1700000000,0))
 if !strings.Contains(endpoint,"timestamp=1700000000000")||!strings.Contains(endpoint,"sign="){t.Fatal("missing DingTalk signature")}
}
