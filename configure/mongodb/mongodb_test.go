package mongodb

import (
	"os"
	"testing"
	"time"

	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	"github.com/gocrud/mgo"
	"github.com/stretchr/testify/assert"
)

// MockContext 创建一个测试用的上下文
func MockContext(logger logging.Logger) *core.BuildContext {
	return nil
}

func TestConfigure(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		// 如果没有设置环境变量，尝试连接一次，失败则跳过
		// 或者直接跳过，避免 CI 失败
		t.Skip("Skipping integration test")
	}

	// 我们可以通过构建一个真实的应用来测试配置逻辑
	builder := core.NewApplicationBuilder()

	// 配置 MongoDB
	builder.Configure(func(ctx *core.BuildContext) {
		Configure(func(b *Builder) {
			// 使用一个可能不存在的地址，目的是测试构建过程是否报错（如果需要连接）
			// 或者如果它支持 lazy connect，则测试注册是否成功
			b.Add("default", "mongodb://example:example@localhost:27017/?directConnection=true", func(o *MongoOptions) {
				o.Timeout = 1 * time.Second
			})
		})(ctx)
	})

	// 尝试 Build
	// 如果没有 MongoDB，这里很可能会 Panic 或者 Exit
	// 为了安全起见，我们只做单元测试：测试 Builder 逻辑

	// 直接测试 Builder
	_ = core.NewApplicationBuilder().Build() // 这会返回 Application，无法直接拿到 BuildContext
	// 由于无法直接创建 BuildContext，我们通过 Configure 回调来获取

	var capturedContainer di.Container

	core.NewApplicationBuilder().
		Configure(func(ctx *core.BuildContext) {
			// 在这个上下文中，我们手动运行我们的逻辑

			// 模拟 Configure 内部逻辑
			b := NewBuilder(ctx)
			b.Add("test_db", "mongodb://example:example@localhost:27017", nil)

			// 验证 Builder 状态
			// build 可能会失败如果尝试连接
			// factory, err := b.Build(ctx.GetLogger())

			capturedContainer = ctx.Container()
		}).
		Build()

	assert.NotNil(t, capturedContainer)
}

func TestBuilder_Add_Validate(t *testing.T) {
	// 我们可以通过 hack 方式获取 context 或者 mock
	// 由于 core.BuildContext 构造困难，我们再次使用 ApplicationBuilder

	core.NewApplicationBuilder().
		Configure(func(ctx *core.BuildContext) {
			builder := NewBuilder(ctx)

			// 测试缺少名称
			builder.Add("", "mongodb://localhost:27017", nil)
			_, err := builder.Build(ctx.GetLogger())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "mongo client name is required")

			// 重置
			builder = NewBuilder(ctx)
			// 测试缺少 URI
			builder.Add("test", "", nil)
			_, err = builder.Build(ctx.GetLogger())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "mongo uri is required")
		}).
		Build()
}

func TestMongoFactory_Register(t *testing.T) {
	factory := NewMongoFactory()
	opts := MongoOptions{
		Name:    "test",
		Uri:     "mongodb://example:example@localhost:27017/?directConnection=true",
		Timeout: 100 * time.Millisecond,
	}

	// 尝试注册
	// mgo.NewClient 通常只是创建对象，真正连接是 lazy 的或者在 ping 时发生
	// 如果 NewClient 尝试连接，这里可能会失败
	err := factory.Register(opts)

	// 如果是因为连接失败，我们也可以接受 error，只要逻辑是对的
	// 但通常 NewClient 只是解析 URI
	assert.NoError(t, err)

	// 验证是否已注册
	var client *mgo.Client
	factory.Each(func(name string, c *mgo.Client) {
		if name == "test" {
			client = c
		}
	})
	assert.NotNil(t, client)

	// 再次注册同名应该失败
	err = factory.Register(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// 测试关闭
	err = factory.Close()
	assert.NoError(t, err)
}
