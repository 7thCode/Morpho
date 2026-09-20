// Command example trains an analyzer on a small public-domain corpus, analyzes
// a sentence, and writes the resulting dictionary to dict.json.
//
// The corpus is the opening of three works whose authors died more than 70
// years ago (Natsume Soseki d.1916, Dazai Osamu d.1948), so it can be shipped
// in the repository without licensing questions. It is tiny: the dictionary
// this produces demonstrates the API and is not a production-quality model.
package main

import (
	"fmt"

	"github.com/7thCode/morpho"
)

// corpus: 夏目漱石『吾輩は猫である』『坊っちゃん』、太宰治『走れメロス』の冒頭。
const corpus = `吾輩は猫である。名前はまだ無い。
どこで生れたかとんと見当がつかぬ。何でも薄暗いじめじめした所でニャーニャー泣いていた事だけは記憶している。
吾輩はここで始めて人間というものを見た。

親譲りの無鉄砲で小供の時から損ばかりしている。
小学校に居る時分学校の二階から飛び降りて一週間ほど腰を抜かした事がある。

メロスは激怒した。必ず、かの邪智暴虐の王を除かなければならぬと決意した。
メロスには政治がわからぬ。メロスは、村の牧人である。
笛を吹き、羊と遊んで暮して来た。けれども邪悪に対しては、人一倍に敏感であった。`

func main() {
	// Start from an empty dictionary so that every run produces the same dict.json
	// (Train accumulates on top of whatever New would load from disk).
	analyzer := morpho.NewInMemory()

	if err := analyzer.Train(corpus); err != nil {
		panic(err)
	}

	results, err := analyzer.Analyze("吾輩は東京で2024年に猫を見た。")
	if err != nil {
		panic(err)
	}

	fmt.Println("形態素解析結果:")
	for _, m := range results {
		fmt.Printf("  %-12s %s\n", m.Surface, m.POS)
	}

	if err := analyzer.Save("dict.json"); err != nil {
		panic(err)
	}

	fmt.Println("\n辞書を dict.json に保存しました")
}
