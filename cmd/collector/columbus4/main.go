package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	fcd_collector "github.com/dezswap/cosmwasm-etl/collector/terra/fcd"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/dezswap/cosmwasm-etl/pkg/dex/terraswap"
	"github.com/dezswap/cosmwasm-etl/pkg/logging"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/fcd"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var columbus4_pairs = []string{
	"terra1a5cc08jt5knh0yx64pg6dtym4c4l8t63rhlag3", "terra1u2g4fc0k4tq6z6lrdwhm4gry3q55dg8k9anjtw", "terra1g7an9lfz22gkv74238dhc2j6ymfp4jy0k55yxk", "terra1xlgl3xvkha2y6mssy9s4qe70sq295825sdmt2q", "terra1uenpalqlmfaf4efgtqsvzpa3gh898d9h2a232g",
	"terra170lzdyflaamashcqkst23k9ew773dtg67tfu5m", "terra1yngadscckdtd68nzw5r5va36jccjmmasm7klpp", "terra1dq27eeasl3rtlc8j3ls5yz8wurnksahj240tsr", "terra13yc7dcphaxpgd538msys7r75d3lwa5mu9u6d88", "terra1prfcyujt9nsn5kfj5n925sfd737r2n8tk5lmpv",
	"terra1u475wh425cs3wmgjqn4fqxyqpv7qmsnw7qs0sd", "terra1xxrx7rgpzlep532ulyyndpm7g2md3zspgepmfa", "terra1seh55ualnngk9jau3qsw7xrl4q7vqhc8y64w5q", "terra17rvtq0mjagh37kcmm4lmpz95ukxwhcrrltgnvc", "terra1etdkg9p0fkl8zal6ecp98kypd32q8k3ryced9d",
	"terra1krny2jc0tpkzeqfmswm7ss8smtddxqm3mxxsjm", "terra1nsdxqpe5upkvy9jegqdkzyhetravf8vl7m496h", "terra1n7kaxpxslz5aw36gx2mnc9a30n4yu0us7x2xj7", "terra1h205tvn624npz49snyha3l4cv60d9kccvdr3cn", "terra1ps8wcwxt783qq07gfza0f67xf3kuaguq0r0e4y",
	"terra1lr6rglgd50xxzqe6l5axaqp9d5ae3xf69z3qna", "terra1h7t2yq00rxs8a78nyrnhlvp0ewu8vnfnx5efsl", "terra1sa70wy0857fzu2gjcrg4mm7n5767k7lhj3yrct", "terra18cxcwv0theanknfztzww8ft9pzfgkmf2xrqy23", "terra1gq7lq389w4dxqtkxj03wp0fvz0cemj0ek5wwmm",
	"terra1u4je05rr7galjya4pn2c6s4gutg4evmt9qlxl5", "terra1r9e7efml7zk8fd5x5atk39e8jua9kdt7n7lp5r", "terra1gg0snqxxxhjcma4095nn7lduwtv3k5xw34lzks", "terra19073h5d7rczrnu49r8aje5yly5llenv878eekm", "terra1wm0ht8upyxy6lkn236hmzp657m58wlwsahkgn9",
	"terra10ypv4vq67ns54t5ur3krkx37th7j58paev0qhd", "terra1ahlfw40ruca9h64e2hylwfs75ems76mnfstv4x", "terra1vfk7qfdchgjadv2l2rgmkfnswe5n277kwcjafk", "terra1jvhqxjzezxvss9d0rfpttvz4hv6lp7urvwmx0j", "terra1t4z4tdtn8ka6q5g7x3mx4h5w8ufh07c3s6eca4",
	"terra1aqd837959wjcpzssqqr56rhh7dcq2nrqpgqrdm", "terra10l7zllh9hduam4tugygj9x3f6976auj2xeyegp", "terra1774f8rwx76k7ruy0gqnzq25wh7lmd72eg6eqp5", "terra12mzh5cp6tgc65t2cqku5zvkjj8xjtuv5v9whyd", "terra1haay9t8vvpvdrtd6t9c6pzl2cpr8m4xeadhykv",
	"terra13le0pfhey5ltg42z8e2cq2yt8jxu4ry3tdmwu0", "terra1c5swgtnuunpf75klq5uztynurazuwqf0mmmcyy", "terra1c0afrdc5253tkp5wt7rxhuj42xwyf2lcre0s7c", "terra17v05w2qmmxy864ltjtts0nzqlzw32gp6uynh0y", "terra1xghv9y36n9nfvqmlksj8seysd3fftfgud3zgm4",
	"terra14fyt2g3umeatsr4j4g2rs8ca0jceu3k0mcs7ry", "terra1jxzxyjf2cvl7d88un4pp9dak8tljvxhfpgrqh7", "terra13eclwgxxns9v0m6nxveug7m54v49fsfx9fn88d", "terra1cc2w6lgzxarvawywfr7v4q7nq30tfvnfs62u54", "terra1zw0kfxrxgrs5l087mjm79hcmj3y8z6tljuhpmc",
	"terra1k9ndjyjxmyqe3kk7sq92p42kz8ptrtz9wuyq6n", "terra1l774jntnhr80wsj758fr9hlj49t8j5qf0dxhsg", "terra150scqwurj3m6d5n0yt82zer4w8vsdf6hzl6fcc", "terra1u522uyxu6t9lku7gawf6k8evmrlqkjazgj4yk8", "terra1kcdknzuz8q5zjeyzyy30ykfpf3xs4dzxh57vgw",
	"terra1sndgzq62wp23mv20ndr4sxg6k8xcsudsy87uph", "terra1vs2vuks65rq7xj78mwtvn7vvnm2gn7adjlr002", "terra1tndcaqxkpc5ce9qee5ggqf430mr2z3pefe5wj6", "terra1jxazgm67et0ce260kvrpfv50acuushpjsz2y0p", "terra15g3ecsr7xmz8q9uhtc0r340uj9eut4pg3umc9w",
	"terra1afdz4l9vsqddwmjqxmel99atu4rwscpfjm4yfp", "terra1ml720ts5kfg5s9hhsft74hy6clqsevsqkvu0du", "terra15kkctr4eug9txq7v6ks6026yd4zjkrm3mc0nkp", "terra1tn8ejzw8kpuc87nu42f6qeyen4c7qy35tl8t20", "terra19pg6d7rrndg4z4t0jhcd7z9nhl3p5ygqttxjll",
	"terra1mtt2dpjah3mja54rg9exyexsdkw8u3pljdu34j", "terra1myn9wsgv02jny8jwhnxdf8lrfy6gvdnv2c3wk3", "terra108ukjf6ekezuc52t9keernlqxtmzpj4wf7rx0h", "terra1uqju4mlp0f000atx07xd49y3dlwe50e0d8d4xe", "terra1yppvuda72pvmxd727knemvzsuergtslj486rdq",
	"terra1zktjl62e2hsv2h8mcfs856gu9tjgy58fk6wn6a", "terra1n6e0tr9f2ygkjthy56ckuhxrzs0xa5jmtw7guu", "terra1amv303y8kzxuegvurh0gug2xe9wkgj65enq2ux", "terra18adm0emn6j3pnc90ldechhun62y898xrdmfgfz", "terra1q2cg4sauyedt8syvarc8hcajw6u94ah40yp342",
	"terra1gm5p3ner9x9xpwugn9sp6gvhd0lwrtkyrecdn3", "terra1pdxyk2gkykaraynmrgjfq2uu7r9pf5v8x7k4xk", "terra1f22zcf6m4600eajay6wnzw03lmjdy0f8mjxphe", "terra1q4xywudqrgsuvdc0n9w0mhmrpzqj9lrl5wcf6j", "terra1hl4624c6pkza6jyxnzpyxf07vyhuwt34c67ujf",
	"terra1pxpdtcydzh59rxwdnz9prs92emws7knz5gmugj", "terra1f6d9mhrsl5t6yxqnr4rgfusjlt3gfwxdveeyuy", "terra178jydtjvj4gw8earkgnqc80c3hrmqj4kw2welz", "terra1t7zq9ujczprlqss0dec7akfmnc2wumvsl8dr2v", "terra1u56eamzkwzpm696hae4kl92jm6xxztar9uhkea",
	"terra1nsmppls52m3fmphznag3f7lue5wp6ffrp9e4s5", "terra1p6nzgyy7gq9hw54errcyrnjkf7p4expcsnmxz3", "terra1dkc8075nv34k2fu6xn6wcgrqlewup2qtkr4ymu", "terra1vnrd37f7cjwgu9zgn7whqcjjuclhhd38qrnqlw", "terra1he8as5az55glkg7rye69ujsgl8f8gs3khjxwm2",
	"terra1ea9js3y4l7vy0h46k4e5r5ykkk08zc3fx7v4t8", "terra1pn20mcwnmeyxf68vpt3cyel3n57qm9mp289jta", "terra1ldpgn35p9m2ksumh8dksp3edvv4nz9tdj2pwv2", "terra1gwd9m3e08m53fnhgv8k9vsjrmvwelhpxr2ww8h", "terra1fhgksqjyawsym7myq04ddvm0sgxwzqqlw4pp5j",
	"terra1vkvmvnmex90wanque26mjvay2mdtf0rz57fm6d", "terra1yl2atgxw422qxahm02p364wtgu7gmeya237pcs", "terra17eakdtane6d2y7y6v0s79drq7gnhzqan48kxw7", "terra1mp40g7sneygtnxdm2ncxmjq45pgut2hryxck3a", "terra1u3pknaazmmudfwxsclcfg3zy74s3zd3anc5m52",
	"terra1pc0r79hmaznqh7tnsmzmh9zsl5y87t02zz2fmr", "terra19uem3h0laypj9w0u64gaz782wqe0s909uaarkp", "terra1nx3rggxrpv0yekes85kg3fa7v73qgphst6v5gj", "terra12zwtkdu6qhs4f8jw9zvdgrwl24mwag5kuh7gus", "terra14hklnm2ssaexjwkcfhyyyzvpmhpwx6x6lpy39s",
	"terra16kgssn83sz92u22mmqtgmd795edtttu7apjsm9", "terra1zz39wfyyqt4tjz7dz6p7s9c8pwmcw2xzde3xl8", "terra1urnd4esuqwq89ylqrrh6lktp3rdaew9hqm8txc", "terra1n3khk8w48y8jf5cgjjhccg5dg9qua0xdkt98we", "terra1twvmq2yfahhdpelavz94d9xhcmrfyazrn08lrz",
	"terra1zey9knmvs2frfrjnf4cfv4prc4ts3mrsefstrj", "terra1ze5f2lm5clq2cdd9y2ve3lglfrq6ap8cqncld8", "terra1wrwf3um5vm30vpwnlpvjzgwpf5fjknt68nah05",
}

// TerraSwap Columbus-4 network-specific transaction collector from the FCD server.
func main() {
	cfg := configs.New().Collector.FcdConfig
	dbCon := cfg.RdbConfig
	dbDsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbCon.Host, dbCon.Port, dbCon.Username, dbCon.Password, dbCon.Database)
	writer := io.MultiWriter(os.Stdout)
	db, err := gorm.Open(postgres.Open(dbDsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: glogger.New(
			log.New(writer, "\r\n", log.LstdFlags),
			glogger.Config{
				IgnoreRecordNotFoundError: true,
				SlowThreshold:             time.Second,
				Colorful:                  false,
				LogLevel:                  glogger.Warn,
			},
		),
	})
	if err != nil {
		panic(err)
	}
	store := fcd_collector.NewPermanentStore(db)

	fcdIns := fcd.New(cfg.Url, &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:      10,               // Maximum idle connections to keep open
			IdleConnTimeout:   30 * time.Second, // Time to keep idle connections open
			DisableKeepAlives: false,            // Use HTTP Keep-Alive
		},
	})
	repo := fcd_collector.NewFcdRepo(fcdIns)

	// default setting is for columbus-4 network
	targets := columbus4_pairs
	cfg.UntilHeight = terraswap.COLUMBUS_4_END_HEIGHT

	app := fcd_collector.New(repo, store)
	logger := logging.New("col4_collector", configs.Get().Log)
	defer catch(logger)

	errTolerance := 3
	wait := 3 * time.Second
	for _, addr := range targets {
		logger.Infof("start collecting addr(%s)", addr)
		for errCount := uint(0); errCount <= uint(errTolerance); {
			if err := app.Collect(addr, uint32(cfg.UntilHeight)); err != nil {
				errCount++
				logger.Errorf("errCount: %d, err: %s", errCount, err)
			} else {
				break
			}
			wait := wait * time.Duration(math.Pow(2, float64(errCount)))
			time.Sleep(wait)
		}
		logger.Infof("finished collecting addr(%s)", addr)
	}
}

func catch(logger logging.Logger) {
	recovered := recover()

	if recovered != nil {
		defer os.Exit(1)

		err, ok := recovered.(error)
		if !ok {
			logger.Errorf("could not convert recovered error into error: %s\n", spew.Sdump(recovered))
			return
		}

		stack := string(debug.Stack())
		logger.WithField("err", logging.NewErrorField(err)).WithField("stack", stack).Errorf("panic caught")
	}
}
