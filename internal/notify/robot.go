package notify

import(
 "bytes"
 "context"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "strings"
 "time"
)
func isRobotProvider(p WebhookProvider)bool{return p==WebhookWeCom||p==WebhookDingTalk||p==WebhookFeishu}
func robotEndpoint(p WebhookProvider,secret string)(string,error){
 u,err:=url.Parse(strings.TrimSpace(secret))
 invalid:=fmt.Errorf("机器人 Webhook 地址无效，请复制对应平台的完整 HTTPS 地址")
 if err!=nil||u.Scheme!="https"||u.User!=nil||u.Fragment!=""||u.Port()!=""{return "",invalid}
 switch p{
 case WebhookWeCom:if u.Host!="qyapi.weixin.qq.com"||u.Path!="/cgi-bin/webhook/send"||u.Query().Get("key")==""{return "",invalid}
 case WebhookDingTalk:if u.Host!="oapi.dingtalk.com"||u.Path!="/robot/send"||u.Query().Get("access_token")==""{return "",invalid}
 case WebhookFeishu:if u.Host!="open.feishu.cn"||!strings.HasPrefix(u.Path,"/open-apis/bot/v2/hook/")||strings.TrimPrefix(u.Path,"/open-apis/bot/v2/hook/")==""{return "",invalid}
 default:return "",invalid
 }
 return u.String(),nil
}
func sendRobot(ctx context.Context,client *http.Client,cfg RuntimeConfig,content AlertContent)error{
 primary,signature:=splitRobotCredential(webhookSecret(cfg))
 endpoint,err:=robotEndpoint(cfg.Webhook.Provider,primary);if err!=nil{return err}
 message:=content.PlainBody
 // Ensure UTF-8 and bounded message length for group robots.
 if len(message)>1800{end:=1800;for end>0&&(message[end]&0xc0)==0x80{end--};message=message[:end]+"…"}
 var payload map[string]any
 if cfg.Webhook.Provider==WebhookFeishu{payload=map[string]any{"msg_type":"text","content":map[string]string{"text":message}}}else{payload=map[string]any{"msgtype":"text","text":map[string]string{"content":message}}}
 if cfg.Webhook.Signed {if signature==""{return fmt.Errorf("请填写机器人签名密钥")};endpoint=signRobot(cfg.Webhook.Provider,endpoint,signature,payload,time.Now())}
 body,err:=json.Marshal(payload);if err!=nil{return err}
 req,err:=http.NewRequestWithContext(ctx,http.MethodPost,endpoint,bytes.NewReader(body));if err!=nil{return fmt.Errorf("机器人请求构造失败")}
 req.Header.Set("Content-Type","application/json")
 if client==nil{client=&http.Client{Timeout:10*time.Second}}
 local:=*client;local.CheckRedirect=func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}
 resp,err:=local.Do(req);if err!=nil{return fmt.Errorf("机器人请求失败，请检查网络或 Webhook 配置")};defer resp.Body.Close()
 if resp.StatusCode<200||resp.StatusCode>=300{return fmt.Errorf("机器人返回 HTTP %d",resp.StatusCode)}
 var result struct{ErrCode *int `json:"errcode"`;Code *int `json:"code"`;StatusCode *int `json:"StatusCode"`}
 if err:=json.NewDecoder(io.LimitReader(resp.Body,8192)).Decode(&result);err!=nil{return fmt.Errorf("机器人响应无效")}
 code:=result.ErrCode
 if cfg.Webhook.Provider==WebhookFeishu{code=result.Code;if code==nil{code=result.StatusCode}}
 if code==nil{return fmt.Errorf("机器人响应缺少状态码")}
 if *code!=0{return fmt.Errorf("机器人拒绝消息（错误码 %d），请检查地址与群机器人安全设置",*code)}
 return nil
}
