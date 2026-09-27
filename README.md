# quasi-random-sequence-go

纯本地、可复现的**低差异序列（low-discrepancy / quasi-random sequence）生成与质量评估**库。
目标是在**不增加样本数**的前提下，为数值积分、蒙特卡洛渲染、实验设计等场景提供
覆盖更均匀、维度间相关性更低的单位超立方体点列。

- 无任何网络 / 外部服务依赖，只使用本地 CPU；
- 不使用 Go 运行时的全局随机源，随机化完全由种子决定，**同参数逐位复现**；
- 两种生成机理不同的序列：**Halton**（质数基的根式逆序）与 **Kronecker/Richtmyer**
  （素数平方根小数部分的秩 1 格点序列）；
- 批量与增量生成走同一份无状态计算代码，**结果逐位一致**；
- 每流（stream）独立置乱密钥，可任意并发交错使用；
- 内置确定性的质量评估指标，便于在不同序列间做客观比较。

## 安装 / 要求

- Go 1.25+，仅依赖标准库。

```go
import "github.com/guoyyit/quasi-random-sequence-go/lds"
```

## 快速上手

### 批量生成

```go
// 在 [0,1)^4 中生成 1000 个 Halton 点
pts, err := lds.Generate(4, 1000, lds.Options{Kind: lds.Halton})
```

### 增量生成（单点 / 分块，结果与批量完全一致）

```go
seq, _ := lds.NewSequence(4, lds.Options{Kind: lds.Kronecker})

p1, _ := seq.Next()                 // 一个点 []float64
chunk, _ := seq.NextN(99)           // 一块点 [][]float64
rest, _ := seq.NextN(900)           // Next/NextN 可任意交错
seq.Reset()                         // 回到初始位置
```

### 随机化置乱（保持低差异、可复现）

```go
opts := lds.Options{
    Kind:     lds.Halton,
    Scramble: true,
    Seed:     1234,   // 同种子 + 同参数 => 逐位相同；不同种子 => 不同点列
    Stream:   0,      // 独立子流编号
}
pts, _ := lds.Generate(3, 512, opts)
```

- Halton 使用 **Faure–Tezuka / Owen 式数字置乱**：按数字位置对 b 进制数字做置换，
  连前导零位也参与置乱。每个 b 进基本区间被双射到等体积的另一区间，
  因而保留 (0,m)-网结构与低差异阶，同时消除未置乱 Halton 的原点退化点。
- Kronecker 使用 **Cranley–Patterson 环面平移** `{x + s}`：环面平移不改变差异度，
  且把确定性点列变成同分布的随机化点列（可用于构造置信区间）。

### 质量评估

```go
report, err := lds.Assess(pts, 16) // 16 个等宽网格
fmt.Println(report.L2StarDisc, report.MaxAbsPairCorr, report.ChiSquareMax)
```

也可以单独调用：

- `lds.L2StarDiscrepancy(pts)` —— Warnock–Hyde L2 星差异（O(n²·d)，`n <= L2MaxN=2000` 时计算）；
- `lds.PairCorrelations(pts)` —— 所有维度对的 Pearson 相关系数与最大绝对值；
- `lds.UniformChiSquare(pts, bins)` —— 每维在等概率网格上的 Pearson 卡方统计量。

所有指标都是输入的纯函数，**相同输入结果确定**。报告还包含边界点计数 `EdgeCount`
与越界点计数 `OutOfRange`（越界时返回显式错误，绝不静默）。

## 示例结果

以下数字由本仓库自带示例程序在本机实际产生（Go 版本、平台无关，浮点路径确定）：

```text
$ go run ./examples/qualitydemo -n 1000 -d 3 -bins 16

sequence     L2StarDisc    max|corr|       chiMax      edges   outRange
halton         0.001678       0.0045         0.29          0          0
kronecker      0.002469       0.0195         0.22          0          0
random         0.007596       0.0542        17.02          0          0
```

```text
$ go run ./examples/qualitydemo -n 2000 -d 5 -bins 20

sequence     L2StarDisc    max|corr|       chiMax      edges   outRange
halton         0.001146       0.0027         0.62          0          0
kronecker      0.001954       0.0183         0.30          0          0
random         0.004824       0.0503        21.78          0          0
```

```text
$ go run ./examples/qualitydemo -n 512 -d 2 -bins 16 -scramble -seed 1234

sequence     L2StarDisc    max|corr|       chiMax      edges   outRange
halton         0.001748       0.0014         0.19          0          0
kronecker      0.003203       0.0053         0.38          0          0
random         0.015156       0.0137        18.94          0          0
```

解读：

- **L2 星差异**：越小越均匀。两条 LDS 序列比 i.i.d. 均匀随机低约 3–6 倍；
- **最大维度间相关系数**：接近 0，说明各维几乎不相关；
- **卡方最大值**：随机样本约 17–22，而 LDS 远小于 1，说明网格填充非常均匀；
- **edges / outRange**：所有点严格落在开立方体 (0,1) 内。

运行自己的对比：

```bash
go run ./examples/qualitydemo -n 1000 -d 3 -bins 16
go run ./examples/qualitydemo -n 512 -d 2 -scramble -seed 1234
```

## 设计要点

### 索引约定（无退化点）

公开第 k 个点（k=0,1,2,…）对应生成器整数索引 `StartIndex + k + 1`。
从 1 开始是为了跳过两种序列在整数索引 0 处共同的全零原点
（Halton 的 0 根式逆序、Kronecker 的 {0·α}），保证所有返回点都在 (0,1) 内。

### Kronecker 的大索引精度

朴素写法 `frac(i*alpha)` 在大 i 时会发生灾难性抵消。本实现把 α 一次性量化为
64 位定点数 Q = round(α·2^64)，然后用 `math/bits` 的精确 128 位乘法取低 64 位：

```
{i·α} ≈ ((i·Q) mod 2^64) / 2^64
```

单点 O(1)、**无累积状态**，因此增量与批量路径执行完全相同的代码，天然逐位一致。

### 并发模型

- `Generate` 是无状态纯函数，可任意并发调用相同参数，结果恒等；
- `Sequence` 是有状态游标，**每个 goroutine 应各自创建一个**。不同 `Stream`
  的置乱密钥由 `(Seed, Stream, 维度/位置)` 经 SplitMix64 派生，相互独立；
  任意交错调用与顺序调用结果相同，流之间不产生重复点（测试会逐点校验）。

随机源是固定规范的 **SplitMix64**（`lds/rng.go`），不依赖 runtime 随机数，
因此输出在 Go 版本与平台之间保持一致。

## 参数校验与错误

错误均为可 `errors.Is` 判定的哨兵错误，绝不 panic、绝不静默返回非法点：

| 情况 | 错误 |
| --- | --- |
| `n <= 0`（零 / 负样本） | `ErrInvalidSamples` |
| `dim <= 0`（零 / 负维度） | `ErrInvalidDimension` |
| `dim > MaxDim`（=1024，超大维度） | `ErrInvalidDimension` |
| `Stream < 0` | `ErrInvalidSeed` |
| `StartIndex < 0` | `ErrInvalidSamples` |
| 绝对索引超过 2^63−1（溢出风险） | `ErrTooLarge` |
| 未知序列类型 | `ErrUnsupported` |
| 评估空点集 / 锯齿矩阵 / 非有限或越界坐标 | `ErrInvalidSamples` / `ErrInvalidDimension` |

## API 速览

```go
const MaxDim = 1024

func Generate(dim, n int, opts Options) ([][]float64, error)
func NewSequence(dim int, opts Options) (Sequence, error)

type Sequence interface {
    Next() ([]float64, error)
    NextN(n int) ([][]float64, error)
    Reset()
    Dim() int
    Index() int64
}

type Options struct {
    Kind       Kind   // lds.Halton（默认）或 lds.Kronecker
    Scramble   bool
    Seed       uint64
    Stream     int64
    StartIndex int64
}

func Assess(points [][]float64, bins int) (*QualityReport, error)
```

## 测试

```bash
go test ./...            # 全部测试
go test ./lds/ -race     # 含并发竞态检测
go test ./... -v         # 查看每个用例
```

覆盖范围：

- **可复现性**：重复批量调用、同种子置乱逐位相等；不同种子点列不同；
- **增量 / 批量一致性**：单次 `NextN`、逐点 `Next`、分块交错均与 `Generate` 相同；
  `Reset` 后可重现；
- **并发独立性**：多流在 goroutine 中交错生成，与各流顺序结果逐点相等，
  且流间无重复点（`-race` 通过）；
- **数学已知值**：未置乱 van der Corput、首个 Halton 点、Richtmyer 首坐标；
- **质量比较**：两条 LDS 的 L2 星差异优于固定种子的 i.i.d. 均匀随机；
- **异常边界**：零/负样本、零/负/超大维度、越界与非有限坐标、锯齿矩阵、
  非法 bins、2^63 索引溢出、超大 n（L2 计算跳过并返回哨兵值）。
