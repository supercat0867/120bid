package http

import (
	"fmt"
	"testing"
)

func TestCleanAnnouncementHTML(t *testing.T) {
	input := `
<article class="content" id="content">
            <div>
                <p>澄清或变更简要说明：本项目投标截止时间和开标时间变更到2025年10月30日上午10点00分（北京时间）。特此公告！<br />
中航技国际经贸发展有限公司<br />
中航技国际经贸发展有限公司受招标人委托对下列产品及服务进行国际公开竞争性招标，于2025-09-19在中国国际招标网发布变更公告。本次招标采用传统招标方式，现邀请合格投标人参加投标。<br />
1、招标条件<br />
项目概况:<a href="/users/wxsndlyxgs.html">无锡深南电路有限公司</a>项目<br />
资金到位或资金来源落实情况:已落实<br />
项目已具备招标条件的说明:已具备<br />
2、招标内容<br />
招标项目编号:0730-244010SZ0082/06<br />
招标项目名称:<a href="/users/wxsndlyxgs.html">无锡深南电路有限公司</a>项目-皮秒激光切割机【重新招标】<br />
项目实施地点:中国江苏省<br />
招标产品列表(主要设备):<br />
3、投标人资格要求<br />
投标人应具备的资格或业绩:投标人应是响应招标、已在招标机构处领购招标文件，并参加投标竞争的法人或其他组织。凡是来自中华人民共和国或是与中华人民共和国有正常贸易往来的国家或地区的法人或其他组织均可投标。<br />
是否接受联合体投标:不接受<br />
未领购招标文件是否可以参加投标:不可以<br />
4、招标文件的获取<br />
招标文件领购开始时间:2024-12-12<br />
招标文件领购结束时间:2025-10-29<br />
是否在线售卖标书:否<br />
获取招标文件方式:现场领购<br />
招标文件领购地点：中航招标网数智招采运营平台（https://www.avicbid.com/）<br />
招标文件售价:￥500/$100<br />
其他说明:凡有意参加本项目者,请登录中航招标网数智招采运营平台（https://www.avicbid.com/）完成注册（无需办理付费会员），然后选择相应的项目缴纳标书费购买文件。<br />
提示：①免费注册流程：访问“中航招标网”（https://www.avicbid.com）点击上方【注册】-【供应商注册】-录入供应商信息并提交审核（已注册过的投标人直接进入“报名流程”）；②报名流程：访问“中航招标网”（https://www.avicbid.com/）点击上方【登录】-【我要报名】-【搜索项目】-【报名】-填写报名信息；③标书费缴纳流程：在供应商主界面点击【项目管理】-【我参与的项目】-【标书费】-【购物车勾选缴纳标书费】-【提交订单】-完成支付。④支付成功后，把【订单】界面截图发送至招标代理联系人的工作邮箱（yyp@aitedsz.cn）。注册、报名及缴费问题可联系咨询电话：4006-722-788，北京时间：上午9：00～11：30，下午2：00～5：00时(节假日除外)，或查询网站操作指引：“中航招标网”首页【帮助中心】-【下载专栏】。<br />
5、投标文件的递交<br />
投标截止时间（开标时间）:2025-10-30 10:00<br />
投标文件送达地点:深圳市福田区华富路南光大厦3楼301室<br />
开标地点:深圳市福田区华富路南光大厦3楼301室<br />
6、联系方式<br />
招标人:<a href="/users/wxsndlyxgs.html">无锡深南电路有限公司</a><br />
地址:深圳市龙岗区坪地街道盐龙大道1639号<br />
联系人:谭东昱<br />
联系方式:0755-89300000-20717<br />
招标代理机构:中航技国际经贸发展有限公司<br />
地址:北京朝阳区慧忠路5号远大中心B座20层<br />
联系人:温尧尧、孙雪娟、马闯<br />
联系方式:0755-25322736、0755-83663706、yyp@aitedsz.cn<br />
7、汇款方式:<br />
招标代理机构开户银行(人民币):<br />
招标代理机构开户银行(美元):<br />
账号(人民币):<br />
账号(美元):</p>
<li>附件1：<span class="span-link" data-url="aHR0cDovL2ltYWdlcy5jaGluYWJpZGRpbmcubW9mY29tLmdvdi5jbi9qZ2ZpbGUvZnRwL3pidy9jbGllbnQvYmlsaWFuL2JpbGlhbkZpbGUvNDA3MTIwLzIwMjUvMDkvZWM4YjA3YjY1YTk5NDQ3ZGI5MzZjZTczMGYyMjU3ZmYvJUU2JThCJTlCJUU2JUEwJTg3JUU1JTk1JTg2JUU1JTkzJTgxMDA4MjA2Lnhscw==" title="点击打开链接">招标商品008206.xls</span></li>
            </div>
            <p>本公告地址：https://www.120bid.com/view/16042/3q4kYJkBg7S5K-63bhq8.html</p>
            <div class="view-url"><a href="https://www.120bid.com/view/16042/3q4kYJkBg7S5K-63bhq8.html" data-view="aHR0cHM6Ly9jaGluYWJpZGRpbmcubW9mY29tLmdvdi5jbi9iaWREZXRhaWwvL2JpZGRpbmcvYnVsbGV0aW4vMjAyNTA5L2ZmODA4MDgxOTgwNjgzNzYwMTk5NWZmOGM2MGQzYjFkLmh0bWw=" target="_blank" rel="external nofollow noopener noreferrer">公告原网站链接</a></div>
            <div class="social-share" style="text-align: center" data-disabled="tencent,google,twitter,facebook,linkedin,diandian"></div>
        </article>`

	got, err := CleanAnnouncementHTML(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fmt.Println(got)
}
