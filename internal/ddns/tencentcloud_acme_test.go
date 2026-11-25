package ddns

import (
	"context"
	"fmt"
	"testing"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

type fakeTencentTXT struct {
	records []*dnspod.RecordListItem
	creates int
}
func (f *fakeTencentTXT) DescribeRecordListWithContext(ctx context.Context,r *dnspod.DescribeRecordListRequest)(*dnspod.DescribeRecordListResponse,error){
	if err:=ctx.Err();err!=nil{return nil,err}
	if *r.Subdomain!="_acme-challenge"||*r.RecordType!="TXT"{return nil,fmt.Errorf("wrong challenge query")}
	start:=int(*r.Offset)
	end:=start+int(*r.Limit)
	if end>len(f.records){end=len(f.records)}
	resp:=dnspod.NewDescribeRecordListResponse()
	resp.Response=&dnspod.DescribeRecordListResponseParams{RecordList:f.records[start:end]}
	return resp,nil
}
func (f *fakeTencentTXT) CreateRecordWithContext(ctx context.Context,r *dnspod.CreateRecordRequest)(*dnspod.CreateRecordResponse,error){
	if r.TTL!=nil{return nil,fmt.Errorf("TTL must be omitted")}
	if *r.SubDomain!="_acme-challenge"||*r.RecordType!="TXT"{return nil,fmt.Errorf("wrong TXT name")}
	f.creates++
	f.records=append(f.records,&dnspod.RecordListItem{RecordId:common.Uint64Ptr(uint64(1000+f.creates)),Value:r.Value})
	return dnspod.NewCreateRecordResponse(),nil
}
func (f *fakeTencentTXT) DeleteRecordWithContext(ctx context.Context,r *dnspod.DeleteRecordRequest)(*dnspod.DeleteRecordResponse,error){
	for i,rec:=range f.records {if *rec.RecordId==*r.RecordId{f.records=append(f.records[:i],f.records[i+1:]...);return dnspod.NewDeleteRecordResponse(),nil}}
	return nil,fmt.Errorf("unexpected deletion")
}
func TestTencentApexWildcardTXT(t *testing.T){
	ctx:=context.Background()
	f:=&fakeTencentTXT{}
	zone:=&dnspod.DomainListItem{Name:common.StringPtr("example.cn")}
	for i:=0;i<101;i++{f.records=append(f.records,&dnspod.RecordListItem{RecordId:common.Uint64Ptr(uint64(i+1)),Value:common.StringPtr(fmt.Sprintf("unrelated-%d",i))})}
	for _,v:=range []string{"apex","wildcard","apex"}{if err:=ensureTencentTXT(ctx,f,zone,"_acme-challenge",v);err!=nil{t.Fatal(err)}}
	if f.creates!=2||len(f.records)!=103{t.Fatal("expected two distinct records and idempotent retry")}
	if err:=deleteTencentTXT(ctx,f,zone,"_acme-challenge","apex");err!=nil{t.Fatal(err)}
	if len(f.records)!=102||*f.records[101].Value!="wildcard"{t.Fatal("cleanup removed another challenge")}
	if err:=deleteTencentTXT(ctx,f,zone,"_acme-challenge","wildcard");err!=nil{t.Fatal(err)}
	if len(f.records)!=101{t.Fatal("unrelated records changed")}
	if err:=deleteTencentTXT(ctx,f,zone,"_acme-challenge","");err==nil{t.Fatal("empty cleanup accepted")}
}
