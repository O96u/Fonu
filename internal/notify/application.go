package notify

import (
 "bytes"
 "context"
 "crypto/hmac"
 "crypto/sha256"
 "encoding/base64"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "strconv"
 "strings"
 "sync"
 "time"
)

const robotCredentialPrefix="fonu-robot-v1:"
type robotCredential struct {Primary string `json:"primary"`; Signature string `json:"signature"`}
func splitRobotCredential(raw string)(string,string){
 if strings.HasPrefix(raw,robotCredentialPrefix){var v robotCredential;if json.Unmarshal([]byte(strings.TrimPrefix(raw,robotCredentialPrefix)),&v)==nil{return v.Primary,v.Signature};return "",""}
 return raw,""
}
func webhookSecretFromRuntime(cfg RuntimeConfig)string{return webhookSecret(cfg)}
func appProvider(p WebhookProvider)bool{return p==WebhookWeComApp||p==WebhookFeishuApp}
func sameWebhookIdentity(a,b WebhookConfig)bool{return a.Provider==b.Provider&&(!appProvider(a.Provider)||(strings.TrimSpace(a.AppID)==strings.TrimSpace(b.AppID)&&strings.TrimSpace(a.APIBase)==strings.TrimSpace(b.APIBase)))}
func resolveWebhookCredential(c WebhookConfig,input string,old WebhookConfig,stored string)(string,error){
 primary,sign:=splitRobotCredential(strings.TrimSpace(input))
 same:=sameWebhookIdentity(c,old)
 oldPrimary,oldSign:=splitRobotCredential(stored)
 if primary==""||primary==MaskedSecret{primary="";if same{primary=oldPrimary}}
 if isRobotProvider(c.Provider)&&c.Signed {
  if c.Provider==WebhookWeCom{return "",fmt.Errorf("企业微信群机器人不支持此签名模式")}
  if sign==""||sign==MaskedSecret{sign="";if same&&old.Signed{sign=oldSign}}
  if primary==""||sign==""{return "",fmt.Errorf("请填写机器人地址和签名密钥")}
  raw,_:=json.Marshal(robotCredential{primary,sign});return robotCredentialPrefix+string(raw),nil
 }
 if !isRobotProvider(c.Provider)&&strings.HasPrefix(input,robotCredentialPrefix){return "",fmt.Errorf("通知凭据格式无效")}
 return primary,nil
}
func appBase(c WebhookConfig)(string,error){
 if c.Provider==WebhookFeishuApp{return "https://open.feishu.cn",nil}
 base:=strings.TrimRight(strings.TrimSpace(c.APIBase),"/");if base==""{return "https://qyapi.weixin.qq.com",nil}
 u,e:=url.Parse(base);if e!=nil||u.Scheme!="https"||u.Hostname()==""||u.User!=nil||u.RawQuery!=""||u.Fragment!=""{return "",fmt.Errorf("企业微信转发地址必须为有效 HTTPS 地址")}
 return base,nil
}
func validateWebhookCredential(c WebhookConfig,raw string)error{
 if isRobotProvider(c.Provider){primary,sign:=splitRobotCredential(raw);if _,err:=robotEndpoint(c.Provider,primary);err!=nil{return err};if c.Signed&&(sign==""||c.Provider==WebhookWeCom){return fmt.Errorf("签名配置无效")}}
 if appProvider(c.Provider){
  if strings.TrimSpace(c.AppID)==""||strings.TrimSpace(raw)==""||raw==MaskedSecret||strings.TrimSpace(c.Recipient)==""{return fmt.Errorf("请填写应用 ID、Secret 和接收对象")}
  if c.Provider==WebhookWeComApp{if strings.Contains(c.Recipient,"@all"){return fmt.Errorf("请指定接收成员，不支持向全体成员广播")};agent,e:=strconv.ParseInt(c.AgentID,10,32);if e!=nil||agent<=0{return fmt.Errorf("请输入有效的 AgentId")};_,e=appBase(c);return e}
  if c.RecipientType!="open_id"&&c.RecipientType!="chat_id"{return fmt.Errorf("请选择有效的飞书接收对象类型")}
 }
 return nil
}
func signRobot(p WebhookProvider,endpoint,secret string,payload map[string]any,now time.Time)string{
 if p==WebhookFeishu{stamp:=strconv.FormatInt(now.Unix(),10);mac:=hmac.New(sha256.New,[]byte(stamp+"\n"+secret));payload["timestamp"]=stamp;payload["sign"]=base64.StdEncoding.EncodeToString(mac.Sum(nil))}
 if p==WebhookDingTalk{stamp:=strconv.FormatInt(now.UnixMilli(),10);mac:=hmac.New(sha256.New,[]byte(secret));mac.Write([]byte(stamp+"\n"+secret));u,_:=url.Parse(endpoint);q:=u.Query();q.Set("timestamp",stamp);q.Set("sign",base64.StdEncoding.EncodeToString(mac.Sum(nil)));u.RawQuery=q.Encode();return u.String()}
 return endpoint
}
type appResponse struct {ErrCode *int `json:"errcode"`;Code *int `json:"code"`;AccessToken string `json:"access_token"`;TenantToken string `json:"tenant_access_token"`;Expires int `json:"expires_in"`;Expire int `json:"expire"`;InvalidUser string `json:"invaliduser"`}
func appRequest(ctx context.Context,client *http.Client,method,endpoint,bearer string,payload any)(appResponse,error){
 var out appResponse
 var reader io.Reader
 if payload!=nil{b,e:=json.Marshal(payload);if e!=nil{return out,e};reader=bytes.NewReader(b)}
 req,e:=http.NewRequestWithContext(ctx,method,endpoint,reader);if e!=nil{return out,fmt.Errorf("应用请求构造失败")}
 req.Header.Set("Content-Type","application/json");if bearer!=""{req.Header.Set("Authorization","Bearer "+bearer)}
 if client==nil{client=&http.Client{Timeout:10*time.Second}}
 local:=*client;local.CheckRedirect=func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}
 resp,e:=local.Do(req);if e!=nil{return out,fmt.Errorf("应用接口请求失败，请检查网络与配置")};defer resp.Body.Close()
 if resp.StatusCode<200||resp.StatusCode>=300{return out,fmt.Errorf("应用接口 HTTP %d",resp.StatusCode)}
 if e=json.NewDecoder(io.LimitReader(resp.Body,65536)).Decode(&out);e!=nil{return out,fmt.Errorf("应用接口响应无效")}
 return out,nil
}
func appCode(p WebhookProvider,r appResponse)(int,error){code:=r.Code;if p==WebhookWeComApp{code=r.ErrCode};if code==nil{return -1,fmt.Errorf("应用接口缺少状态码")};return *code,nil}
var applicationToken struct {sync.Mutex;key [32]byte;token string;expires time.Time}
func appToken(ctx context.Context,client *http.Client,c WebhookConfig,secret,base string,force bool)(string,error){
 key:=sha256.Sum256([]byte(string(c.Provider)+"\x00"+base+"\x00"+c.AppID+"\x00"+secret))
 applicationToken.Lock();defer applicationToken.Unlock()
 if !force&&applicationToken.key==key&&time.Now().Before(applicationToken.expires){return applicationToken.token,nil}
 var r appResponse;var e error
 if c.Provider==WebhookWeComApp{q:=url.Values{"corpid":{c.AppID},"corpsecret":{secret}};r,e=appRequest(ctx,client,http.MethodGet,base+"/cgi-bin/gettoken?"+q.Encode(),"",nil)}else{r,e=appRequest(ctx,client,http.MethodPost,base+"/open-apis/auth/v3/tenant_access_token/internal","",map[string]string{"app_id":c.AppID,"app_secret":secret})}
 if e!=nil{return "",e};code,e:=appCode(c.Provider,r);if e!=nil{return "",e};if code!=0{return "",fmt.Errorf("应用认证失败（错误码 %d）",code)}
 token,ttl:=r.AccessToken,r.Expires;if c.Provider==WebhookFeishuApp{token,ttl=r.TenantToken,r.Expire};if token==""{return "",fmt.Errorf("应用认证未返回访问凭证")}
 applicationToken.key=key;applicationToken.token=token;applicationToken.expires=time.Now().Add(time.Duration(ttl-60)*time.Second)
 return token,nil
}
func sendApplication(ctx context.Context,client *http.Client,cfg RuntimeConfig,content AlertContent)error{
 c,secret:=cfg.Webhook,webhookSecret(cfg)
 if e:=validateWebhookCredential(c,secret);e!=nil{return e}
 base,e:=appBase(c);if e!=nil{return e}
 for attempt:=0;attempt<2;attempt++{
  token,e:=appToken(ctx,client,c,secret,base,attempt==1);if e!=nil{return e}
  var r appResponse
  if c.Provider==WebhookWeComApp{agent,_:=strconv.Atoi(c.AgentID);r,e=appRequest(ctx,client,http.MethodPost,base+"/cgi-bin/message/send?access_token="+url.QueryEscape(token),"",map[string]any{"touser":c.Recipient,"agentid":agent,"msgtype":"text","text":map[string]string{"content":content.PlainBody}})}else{body,_:=json.Marshal(map[string]string{"text":content.PlainBody});r,e=appRequest(ctx,client,http.MethodPost,base+"/open-apis/im/v1/messages?receive_id_type="+url.QueryEscape(c.RecipientType),token,map[string]string{"receive_id":c.Recipient,"msg_type":"text","content":string(body)})}
  if e!=nil{return e};code,e:=appCode(c.Provider,r);if e!=nil{return e}
  if code==0{if r.InvalidUser!=""{return fmt.Errorf("部分企业微信接收成员无效，请检查可见范围")};return nil}
  if attempt==0&&((c.Provider==WebhookWeComApp&&(code==40014||code==42001))||(c.Provider==WebhookFeishuApp&&(code==99991663||code==99991664||code==99991668))){continue}
  return fmt.Errorf("应用消息发送失败（错误码 %d），请检查权限和接收对象",code)
 }
 return fmt.Errorf("应用访问凭证已失效")
}
