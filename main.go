package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

//自定义中间件 （拦截器）
// /usr/add
// 预处理

func myHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		//通过自定义的中间件设置的值，后续处理只要调用了中间件都能拿到这里的参数
		context.Set("userSession", "userid-1")

		if true {
			context.Next() //满足条件放行
		}

		context.Abort() //不满足停止
	}
}

func main() {

	//gin内置的日志输出到文件
	f, _ := os.Create("gin.log")
	//gin.DefaultWriter = io.MultiWriter(f)
	//如果同时写入文件和控制台可以用
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	//生产环境，不打印debug日志
	gin.SetMode(gin.ReleaseMode)

	ginServer := gin.Default()

	// 初始化 session 中间件
	store := cookie.NewStore([]byte("secret"))
	ginServer.Use(sessions.Sessions("mysession", store))

	//注册中间件
	ginServer.Use(myHandler())

	//加载静态界面
	ginServer.LoadHTMLGlob("edge/static/*")

	//加载资源文件
	//重要，实现静态资源文件服务器
	//第一个 路径是 网站访问得URI， 第二个是当前资源文件路径
	//第二个参数，这里真实存放的目录，可以写绝对路径，也可以写相对于执行程序的路径
	ginServer.Static("/aaa", "./edge/static")

	//文件
	ginServer.StaticFS("/bbb", http.Dir("./edge/static"))

	////响应一个页面给前端(给前端传递数据)
	//ginServer.GET("/edge/console/", func(context *gin.Context) {
	//	context.HTML(http.StatusOK, "index.html", gin.H{
	//		"msg": "这是后端传递过来的数据",
	//	})
	//})

	ginServer.GET("/hello", func(context *gin.Context) {
		context.JSON(200, gin.H{"msg": "hello, world"})

		//返回字符串
		context.String(200, "nihao...")

		//响应 XML
		context.XML(200, gin.H{"msg": "hello, world"})

		//响应 YAML
		context.YAML(200, gin.H{"msg": "hello, world"})

		//响应 html

		//响应 文件

	})

	//接受前段传过来的数据
	//user?userid=123&username=gb123
	ginServer.GET("/user/info", myHandler(), func(context *gin.Context) {
		userid := context.Query("userid")
		username := context.Query("username")

		//取出中间件的设置的值
		session := context.MustGet("userSession").(string)

		context.JSON(http.StatusOK, gin.H{
			"userid":   userid,
			"username": username,
			"session":  session,
		})
	})
	//rest full 形式
	// userinfo/123/gb123
	ginServer.GET("/user/info/:userid/:username", func(context *gin.Context) {
		userid := context.Param("userid")
		username := context.Param("username")
		context.JSON(http.StatusOK, gin.H{
			"userid":   userid,
			"username": username,
		})
	})

	//raw json数据
	ginServer.POST("json", func(context *gin.Context) {
		data, _ := context.GetRawData()
		var m map[string]interface{}
		_ = json.Unmarshal(data, &m)
		context.JSON(http.StatusOK, m)
	})

	// 表单Form
	ginServer.POST("/usr/add", func(context *gin.Context) {
		username := context.PostForm("username")
		password := context.PostForm("password")
		context.JSON(http.StatusOK, gin.H{
			"msg":      "ok",
			"username": username,
			"password": password,
		})
	})

	// 重定向: 301
	ginServer.GET("/test", func(context *gin.Context) {
		context.Redirect(http.StatusMovedPermanently, "http://www.kuangstudy.com")
	})

	// no router
	ginServer.NoRoute(func(context *gin.Context) {
		context.HTML(http.StatusNotFound, "404.html", nil)
	})

	// 路由组
	userGroup := ginServer.Group("/usr")
	{
		userGroup.GET("/add", func(context *gin.Context) {

		})
		userGroup.POST("/login", func(context *gin.Context) {

		})
		userGroup.POST("/logout", func(context *gin.Context) {

		})
	}
	orderGroup := ginServer.Group("order")
	{
		orderGroup.GET("/add")
		orderGroup.DELETE("/delete")
	}

	//这种方式是重定向，URL会跳转，不满足我们需求
	/*
		ginServer.Any("/apps/lw/static/lw-bootstrap/*path", func(c *gin.Context) {
			newPath := "/h5/app-lw-bbtstrap" + c.Param("path")
			c.Redirect(http.StatusMovedPermanently, "http://127.0.0.1:30054"+newPath)
		})
	*/

	ginServer.Any("/", func(context *gin.Context) {
		context.File("./edge/static/index.html")
	})

	ginServer.Any("/edge/console/*path", func(c *gin.Context) {
		c.File("./edge/static/index.html")
	})

	targetURL, _ := url.Parse("http://127.0.0.1:30050")
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	//静态资源妆发
	lwStaticRouter := ginServer.Group("/apps/lw/static/")
	lwStaticRouter.Any("*path", func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/apps/lw/static/lw-bootstrap/") {
			c.Request.URL.Path = "/h5/app-lw-bootstrap" + c.Param("path")
			fmt.Println("c.Request.URL.Path:", c.Request.URL.Path)
			proxy.ServeHTTP(c.Writer, c.Request)
		} else {
			c.Request.URL.Path = c.Param("path")
			fmt.Println("c.Request.URL.Path:", c.Request.URL.Path)
			proxy.ServeHTTP(c.Writer, c.Request)
		}
	})

	//后端API
	lwApiRouter := ginServer.Group("/apps/lw/api/")
	lwApiRouter.Any("*path", func(c *gin.Context) {
		if c.Request.URL.Path == "/apps/lw/api/sign/in" {
			//处理登录
			session := sessions.Default(c)
			session.Set("userid", "123")
			session.Set("username", "gb123")
			session.Save()
			fmt.Println("处理登录成功")
		} else if c.Request.URL.Path == "/apps/lw/api/sign/out" {
			//处理登出
			session := sessions.Default(c)
			session.Clear()
			session.Save()
			fmt.Println("处理登出成功")
		} else if c.Request.URL.Path == "/apps/lw/api/test" {
			//处理
			c.Request.Header.Set("x-auth-token", "123333333333333zzzz")
			c.Request.URL.Path = "/lw2" + c.Param("path")
			proxy.ServeHTTP(c.Writer, c.Request)

		} else {
			c.Request.URL.Path = "/lw2" + c.Param("path")
			fmt.Println("c.Request.URL.Path:", c.Request.URL.Path)
			session := sessions.Default(c)
			value, ok := session.Get("userid").(string)
			if !ok || len(value) == 0 {
				fmt.Println("session error")
			}
			proxy.ServeHTTP(c.Writer, c.Request)
		}
	})

	ginServer.Run(":8082")
}
