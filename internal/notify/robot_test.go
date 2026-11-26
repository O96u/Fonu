package notify
import("context";"io";"net/http";"strings";"testing")
type robotTransport func(*http.Request)(*http.Response,error)
func(f robotTransport)RoundTrip(r *http.Request)(*http.Response,error){return f(r)}
func TestRobotProviders(t *testing.T){
 cases:=[]struct{p WebhookProvider;url,body,result string}{
 {WebhookWeCom,"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-secret",`"msgtype":"text"`,`{"errcode":0}`},
 {WebhookDingTalk,"https://oapi.dingtalk.com/robot/send?access_token=test-secret",`"msgtype":"text"`,`{"errcode":0}`},
 {WebhookFeishu,"https://open.feishu.cn/open-apis/bot/v2/hook/test-secret",`"msg_type":"text"`,`{"code":0}`},
 }
 for _,c:=range cases{t.Run(string(c.p),func(t *testing.T){
  result:=c.result
  client:=&http.Client{Transport:robotTransport(func(r *http.Request)(*http.Response,error){b,_:=io.ReadAll(r.Body);if r.URL.String()!=c.url||!strings.Contains(string(b),c.body)||!strings.Contains(string(b),"Fonu"){t.Fatal("invalid robot payload")};return &http.Response{StatusCode:200,Body:io.NopCloser(strings.NewReader(result)),Header:make(http.Header)},nil})}
  cfg:=RuntimeConfig{Config:Config{Webhook:WebhookConfig{Provider:c.p}},WebhookSecret:c.url}
  if err:=SendWebhook(context.Background(),client,cfg,FormatAlert("test","测试","内容"));err!=nil{t.Fatal(err)}
  result=`{"errcode":400,"code":400}`
  if err:=SendWebhook(context.Background(),client,cfg,FormatAlert("test","测试","内容"));err==nil||strings.Contains(err.Error(),"test-secret"){t.Fatal("business error or secret redaction failed")}
  if _,err:=robotEndpoint(c.p,"https://example.com/test-secret");err==nil{t.Fatal("untrusted endpoint")}
 })}
}
