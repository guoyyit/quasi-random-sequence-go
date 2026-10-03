# quasi-random-sequence-go

纯本地、可复现的**低差异序列（quasi-random / low-discrepancy sequence）生成与质量评估**库。
用于在不增加样本数量的前提下，提升数值积分、蒙特卡洛渲染、实验设计等场景在
单位超立方体 `[0,1)^d` 内的覆盖均匀性。

* 不依赖网络、OS 随机源或 CGO；同样的参数永远得到逐位相同的结果。
* 单点增量生成（`Stream.Next`）与批量生成（`Generate`）走同一条确定性
  `pointAt(i)` 路径，因此同一参数下点列完全一致（含 `float64` 逐位一致）。

## 安装与示例

```go
import "github.com/guoyyit/quasi-random-sequence-go/lds"

// 批量生成 400 个 4 维点（Halton）
pts, err := lds.Generate(lds.Config{Kind: lds.Halton, Dim: 4}, 400)

// 随机化置乱：保持低差异性质；同种子逐位复现，不同种子产生不同点列
pts, err = lds.Generate(lds.Config{
    Kind: lds.Halton, Dim: 4, Scramble: true, Seed: 20260927,
}, 400)

// 单点增量（与批量结果一致）
st, _ := lds.NewStream(lds.Config{Kind: lds.Faure, Dim: 4, Scramble: true, Seed: 1})
p, err := st.Next()

// 质量评估（确定性，可用于横向比较不同序列）
m, err := lds.Assess(pts)
fmt.Println(m.L2StarDiscrepancy, m.MaxAbsCorrelation, m.MeanAbsCorrelation)
```

运行演示程序：

```sh
go run ./cmd/lds-demo
```

## 生成机理

| 序列 | 机理 | 说明 |
|------|------|------|
| `lds.Halton` | 不同维度使用不同素数基的**根基逆（radical inverse）** | 支持 1–1000 维（第 1000 个素数 7919） |
| `lds.Faure` | 在不小于维度数的最小素数基上，用 **Pascal 矩阵 `P^j` 的数字网（digital net）** | 与 Halton 构造机理本质不同 |

两类序列都支持置乱。置乱对每个（维度、数位位置、基数）派生一个由
`(Seed, Stream, dim, pos, base)` 决定的 Fisher–Yates 数字置换
（Kocis–Owen 风格的逐坐标数字置乱），保持点列低差异性质，并且：

* 同一 `Seed` 与参数 => 逐位复现；
* 不同 `Seed` => 不同点列；
* `Stream` 编号不同 => 相互独立的置乱子流。

### 严格内部保证

置乱生成的**所有点都严格位于单位超立方体内部**：任何坐标都不等于 0 或 1，
同一点集内不出现重复点，同一种子下不同流的点列互不包含相同点，不同种子
的点列也不相同。批量生成（`Generate`）与单点增量（`Stream.Next`）走同一
条确定性路径，因此该保证对两种生成都成立。

实现上，置乱在逐位置换之外还加入了**无限尾项**（infinite tail）：当一个
点的前导数位被置换为 0 时，尾项仍是严格正的随机值，使坐标不会坍缩到 0，
也使不同下标不会碰撞。尾项按 `(Seed, Stream, dim, pos, base)` 确定性派生
并缓存，因此结果逐位复现，且不影响并发安全。

演示程序 `cmd/lds-demo` 在打印前会对生成结果做**边界与重复校验**：
逐一检查每个坐标严格位于 `(0,1)`，并检查同一点集内无重复点；任何违规都
直接报错退出，绝不继续打印。

## 多流并发

每个 `Stream` 内部互斥，可被多个 goroutine 安全并发调用；`Next` 严格按
逻辑序号返回点，因此“任意交错并发调用”与“顺序调用”得到完全相同的点列。
不同 `Stream`（或不同 `Config.Stream`）状态互不干扰、点列不相关。

```go
st, _ := lds.NewStream(cfg)
var wg sync.WaitGroup
for w := 0; w < 8; w++ {
    wg.Add(1)
    go func() { defer wg.Done(); for { p, err := st.Next(); /* ... */ } }()
}
wg.Wait()
```

## 质量评估指标（均为确定性）

* **L2 星号差异 `L2StarDiscrepancy`**：用 Warnock 闭式公式计算，衡量整体均匀性，
  越小越均匀：
  `D2^2 = 3^{-s} - (2/n)Σ_i Π_d (1-x_id^2)/2 + (1/n^2) Σ_{i,k} Π_d (1-max(x_id,x_kd))`。
  评估复杂度 O(n²)，上限 `maxAssessSamples = 20000`。
* **`MaxAbsCorrelation` / `MeanAbsCorrelation`**：所有坐标列之间 Pearson
  相关系数绝对值的最大值 / 平均值，衡量维度间相关性，越小越好。

### 示例结果（n=400, d=4，本机实际输出）

| 序列 | L2* discrepancy | max\|corr\| | mean\|corr\| |
|------|----------------|-------------|--------------|
| Halton | 0.003832 | 0.009221 | 0.005181 |
| Halton scrambled (seed=1) | 0.003658 | 0.013736 | 0.007045 |
| Faure | 0.003987 | 0.012434 | 0.008164 |
| Faure scrambled (seed=1) | 0.003813 | 0.003515 | 0.002596 |

可以看到低差异序列的星号差异很小，置乱后的 Faure 同时显著降低了维度相关性。

## 参数校验（返回明确错误，绝不崩溃或静默返回非法值）

| 情况 | 返回错误 |
|------|----------|
| 样本数 `n <= 0`（含零样本、负样本） | `ErrInvalidSamples` |
| 样本数超过 `maxGenerateSamples = 10,000,000` | `ErrSamplesTooLarge` |
| 维度 `<= 0`（含零维、负维）或 `> 1000`（超大维度） | `ErrInvalidDim` |
| 未知序列类型 | `ErrInvalidKind` |
| 流编号为负 | `ErrInvalidStreamIndex` |
| 评估空点集 / 零维 / 维数不齐 / 坐标越界或 NaN | `ErrInvalidPoints` |
| 评估样本超过 20000 | `ErrSamplesTooLarge` |

可能造成机器字溢出的组合数（Faure 矩阵元素）使用 `math/big` 精确计算；
每个点坐标以“整数分子 / 整数分母”单次相除得到正确舍入的 `float64`，
该规模下分母不会溢出 `int64`。

## 测试

```sh
go test ./...        # 功能与边界
go test -race ./...  # 并发独立性与数据竞争
```

覆盖：可复现性、增量/批量逐位一致、置乱种子差异、多流并发交错独立性、
点列唯一性、Halton/Faure 已知首点参考值、指标确定性，以及零/负/超大参数边界。

### 置乱专项测试（`lds/scramble_test.go`）

多种子（`1, 2, 42, 12345, 20260927`）× 多维度（`1, 2, 3, 4, 5, 8, 16`）×
两种序列（Halton、Faure）的较大规模扫描，覆盖：

* **严格内部**：所有坐标严格位于 `(0,1)`，不得等于 0 或 1；
* **点集内无重复**：同一点集内任意两点不得相同；
* **跨流无重复**：同种子下不同流的点列不得包含相同点（`dim>=4` 时做逐点
  共享检查，`dim<4` 时断言集合不相同）；
* **跨种子无重复**：不同种子的点列不得包含相同点（同上）；
* **增量/批量逐位一致**：置乱下 `Stream.Next` 与 `Generate` 逐位相同；
* **并发交错一致**：置乱下多 goroutine 交错调用与顺序调用结果一致，
  独立流交错调用互不干扰；
* **低差异保持**：置乱后 L2\* 差异与未置乱处于同一量级（不超过 5 倍），
  维度相关性仍保持很小。

其中“重复点”的验证方法是把每个点按 `fmt.Sprintf("%v", p)` 编码为字符串键
做精确（逐位）比对；“跨流/跨种子独立性”通过检查两个点集是否共享任何点来
验证。测试在 `go test ./...` 与 `go test -race ./...` 下均通过。
