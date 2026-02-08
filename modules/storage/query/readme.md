Query Step 使用模式手册（工程版）

一句话定位
Step = 可组合、可中断、可并行、可复用的「异步函数调用栈」

一、整体心智模型（先对齐）
QueryPlan ≈ 一次函数调用
Step ≈ 函数体中的一条语句
Continuation ≈ return / await
ReturnFromSubPlanStep ≈ return（只返回当前函数）

最重要原则
所有 Step 都只能做一件事

二、核心 Step（必须掌握）
1️⃣ RpcStep ——「异步 IO」
用途

发 RPC

等回调

不存值、不做逻辑

使用场景

跨模块查询

任何可能阻塞的 IO

正确模式
getBlock := &RpcStep{
	Call: func(ctx *QueryContext, cont Continuation) {
		height := ctx.Vars["height"].(uint64)

		ctx.Ctx.InvokeRPC(
			"blockstore",
			"GetByHeight",
			height,
			ctx.RegisterCont(cont),
		)
	},
}

❌ 禁止

在 RpcStep 里解析 / 计算 / 判断

在 RpcStep 里写业务逻辑

2️⃣ ValueStep ——「缓存结果（支柱）」
用途

保存中间结果

后续 Step 通过 ctx.Get() 读取

模式
valBlock := &ValueStep{Inner: getBlock}

block := ctx.Get(valBlock).(*Block)

使用原则

凡是后面要用的结果，必须 ValueStep 包一层

三、逻辑处理类 Step（纯计算）
3️⃣ FuncStep ——「同步逻辑」
用途

计算

组合字段

填充 Vars

数据加工

模式
process := &FuncStep{
	Do: func(ctx *QueryContext, cont Continuation) {
		height := ctx.Get(valHeight).(uint64)
		ctx.Vars["nextHeight"] = height + 1
		cont(nil, nil)
	},
}

特点

同步

不产生值（除非配合 ValueStep）

4️⃣ MapStep ——「值变换」
用途

RPC 返回值 reshape

DTO → domain

类型转换

模式
mapTx := &MapStep{
	From: valRawTx,
	Map: func(v any) (any, error) {
		tx := v.(*RawTx)
		return normalize(tx), nil
	},
}

适合

“只依赖一个输入值”

“没有副作用”

5️⃣ RequireStep ——「断言 / 校验」
用途

判空

合法性检查

提前失败

模式
ensureBlock := &RequireStep{
	From: valBlock,
	Check: func(v any) error {
		if v == nil {
			return ErrBlockNotFound
		}
		return nil
	},
}

设计原则

能用 RequireStep，就不要用 IfStep

四、流程控制类 Step（结构）
6️⃣ SequentialStep ——「顺序执行」
用途

表达「先做什么，再做什么」

&SequentialStep{
	Steps: []Step{
		valHeight,
		valBlock,
		process,
	},
}

规则

最后一个 Step 的返回值 → 上层

7️⃣ ParallelStep ——「并行执行」
用途

多 RPC 并发

聚合结果

&ParallelStep{
	Steps: []Step{
		valA,
		valB,
	},
}

使用建议

并行的 Step 必须互不依赖

常配合 ValueStep 使用

8️⃣ IfStep ——「分支选择」
用途

条件执行

替代 switch / if

&IfStep{
	Cond: func(ctx *QueryContext) bool {
		return ctx.Vars["onlyHeader"].(bool)
	},
	Then: &NoopStep{},
	Else: fillTxs,
}

⚠️ 重要认知

IfStep 不会结束流程

NoopStep ≠ return

五、控制流 / 调用栈类 Step（高级）
9️⃣ ReturnFromSubPlanStep ——「函数 return」
用途

提前结束 当前 SubPlan

不影响外层 QueryPlan

模式
&ReturnFromSubPlanStep{
	Cond: func(ctx *QueryContext) bool {
		return ctx.Vars["block"] == nil
	},
	Value: func(ctx *QueryContext) any {
		return &EmptyBlock{}
	},
	Err: nil,
}

心智模型

return 当前函数，而不是整个程序

🔟 CallStep ——「调用子函数」
用途

复用复杂流程

模拟函数调用

&CallStep{
	Plan: buildGetRpcBlockSubPlan(),
	Bind: func(ctx *QueryContext, v any) {
		ctx.Vars["block"] = v
	},
}

特点

子 Plan 可提前 return

外层继续执行

六、集合处理类 Step
1️⃣1️⃣ ForEachStep ——「循环」
用途

批量 RPC

批量处理

标准模式
&ForEachStep{
	Items: func(ctx *QueryContext) []any {
		return ctx.Vars["hashes"].([]any)
	},
	Step: func(item any) Step {
		return &SequentialStep{
			Steps: []Step{
				setItem(item),
				queryOne,
				collect,
			},
		}
	},
}

子 Step 中取 item
item := ctx.Vars["item"]

七、辅助 / 占位类 Step
1️⃣2️⃣ NoopStep ——「占位」
用途

IfStep 分支占位

可读性增强

Then: &NoopStep{}

1️⃣3️⃣ ResultStep ——「构造最终返回值」
用途

从 ctx.Vars / ctx.Get() 构造结果

&ResultStep{
	Build: func(ctx *QueryContext) any {
		return &QueryResult{
			Data: ctx.Vars["block"],
		}
	},
}

八、常见组合模式（非常重要）
模式 1️⃣：RPC → 校验 → 使用
valBlock := &ValueStep{Inner: getBlock}

ensureBlock := &RequireStep{
	From: valBlock,
	Check: notNil,
}

模式 2️⃣：可提前 return 的通用子流程
&CallStep{
	Plan: buildGetRpcBlockSubPlan(),
	Bind: func(ctx *QueryContext, v any) {
		ctx.Vars["block"] = v
	},
}

模式 3️⃣：复杂 Query 的结构化拆解
SequentialStep{
	Steps: []Step{
		parseArgs,
		getPosition,
		ensurePosition,
		callGetRpcBlock,
		buildTx,
	},
}

九、最后的设计忠告（非常重要）
❗ 你现在这套 Step 框架，不是 ORM，不是 Workflow

它是：

可测试、可复用、可提前返回的异步函数调用系统

永远记住三条铁律：

RPC 永远只放在 RpcStep

共享结果必须 ValueStep

提前结束只用 ReturnFromSubPlanStep