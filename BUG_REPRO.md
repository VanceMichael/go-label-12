# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

生物安全员在解除牛群隔离时，外部合规检查迟迟不返回，于是取消了这次操作。接口却一直挂着，下一位审核员马上重试只得到“该牛群正在审核”；直到原检查服务恢复，两个现象才一起消失。请先不要修改代码，查明取消信号为什么没有结束这次复核，并说明请求、下游检查和群组租约之间的生命周期关系及影响边界。

## 含 Bug 版本

- 仓库：VanceMichael/go-label-12
- 仓库地址：https://github.com/VanceMichael/go-label-12.git
- parent SHA：e3393ceb6662eaae1a18ef74a5c2f022d99a6ab4

## 复现步骤

```bash
git clone -- https://github.com/VanceMichael/go-label-12.git bug-repro
cd bug-repro
git checkout --detach e3393ceb6662eaae1a18ef74a5c2f022d99a6ab4
go test ./internal/compliance -run ^TestCancelledReleaseReviewReturnsAndFreesGroupLease$ -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/compliance -run ^TestCancelledReleaseReviewReturnsAndFreesGroupLease$ -count=1
--- FAIL: TestCancelledReleaseReviewReturnsAndFreesGroupLease (0.11s)
    release_test.go:62: cancelled release review stayed blocked and retained the group lease
FAIL
FAIL	go-base/internal/compliance	0.134s
FAIL

```

stderr：

```text
(empty)
```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/compliance -run ^TestCancelledReleaseReviewReturnsAndFreesGroupLease$ -count=1
--- FAIL: TestCancelledReleaseReviewReturnsAndFreesGroupLease (0.10s)
    release_test.go:62: cancelled release review stayed blocked and retained the group lease
FAIL
FAIL	go-base/internal/compliance	0.102s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

根因结论需要准确指出 ReleaseCoordinator.ReviewRelease 中请求 context 在进入 ReleaseChecker.Check 前被替换的具体行为，并把“取消未唤醒下游检查、函数不能返回、defer 无法释放 tenant/group 租约、下一位审核员获得冲突”串成完整因果链；还应说明外部检查恢复后为何接口与租约会同时恢复，以及不同群组和正常完成路径的边界。用 TestCancelledReleaseReviewReturnsAndFreesGroupLease 的红测作为运行证据，目标仓库代码、测试和配置零改动，不得实施修复或只写成笼统的超时问题。
