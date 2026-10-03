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

### 置乱保证：严格内点、无重复、跨流互不相交

置乱点对**任何种子、任何流编号、任何维度**都满足：

* **严格位于开立方体 `(0,1)^d` 内**：任何坐标都不会等于 `0` 或 `1`，
  批量生成与 `Stream.Next` 单点增量同样成立。可安全用于依赖对数变换
  （如 `log x`）或除法的被积函数。
* **同一序列内无重复点**：在支持的最大样本范围内，任意长度的前序点列
  两两不同。实现上每个坐标都按覆盖最大样本数的**固定位数 K** 逐位做
  双射置换（Halton 不再按索引变长截断），短索引点不会塌缩进粗网格而与
  长索引点重合。
* **跨流不相交**：同一 `Seed` 下不同 `Stream` 的点列不包含相同的点。
  每个置乱点在数字网格分数上额外叠加一个仅由
  `(Seed, Stream, dim, base)` 派生的确定性细网格平移量 `alpha ∈ (0,1)`：
  `x = (A + alpha) / b^K`。平移量是 `2^-m` 的整数倍，且 `b^K * 2^m ≤ 2^51`，
  因此相邻余集之间至少相差两个 `float64` ulp，既保证严格内点，也保证不同
  流/种子的点不可能舍入成同一个 `float64`。
* **未置乱路径零变化**：固定位数只补前导零，有理值 `A/b^K` 不变，
  `float64` 逐位结果与旧版完全一致（参考值测试继续锁定）；配置结构与
  调用方式也没有任何变化。

这些性质的验证方法：用精确的按位键（`math.Float64bits` 序列化，无浮点
容差）统计重复点与跨流重合；边界检查使用严格不等式 `0 < x < 1`；
`go test ./...` 与 `go test -race ./...` 均执行这些检查。

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

置乱专项测试（`lds/scramble_test.go`，两种机理、批量与增量两种路径）：

* 多种子（含 `0` 与 `2^64-1`）× 多维度（1–13 维）扫描：每个坐标严格
  `0 < x < 1`，点集内无按位重复点，且增量与批量逐位一致；
* 50000 点深前缀（跨越多个 `b^k` 位数边界）唯一性回归；
* 同种子 6 个流编号的点集两两不相交；
* 不同种子点列互异，且置乱点列不等于未置乱点列；
* 8 goroutine 并发交错调用与顺序结果一致（按位多重集合校验），两条独立
  流锁步交错互不漂移；
* 置乱/未置乱的 L2 星号差异与最大坐标相关性处于同一量级（不超过 10 倍）。

演示程序 `cmd/lds-demo` 在打印前会对每个点集做边界、重复与跨流重合校验，
发现任何违规直接以非零状态退出，不会打印非法点。
