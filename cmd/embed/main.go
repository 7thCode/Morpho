// Command embed is a proof-of-concept that trains word vectors on top of
// Morpho's own tokenizer output, so the embedding vocabulary always matches
// Morpho's segmentation (unlike a pretrained word2vec model trained with a
// different tokenizer).
package main

import (
	"fmt"
	"os"

	"github.com/7thCode/morpho"
	"github.com/7thCode/morpho/internal/embedding"
	"github.com/7thCode/morpho/internal/hmm"
)

// corpus loosely mixes three topics (animals, food, technology) so that
// co-occurrence statistics give skip-gram something to separate on.
const corpus = `犬は忠実な動物です。猫はかわいい動物です。鳥は空を飛ぶ動物です。
犬と猫はどちらも人気のペットです。動物園には多くの動物がいます。
ラーメンは美味しい食べ物です。寿司も美味しい食べ物です。パンは朝によく食べる食べ物です。
ラーメンと寿司は日本の代表的な食べ物です。美味しい食べ物を食べると幸せです。
コンピュータは便利な機械です。プログラムはコンピュータを動かします。ソフトウェアはプログラムの一種です。
コンピュータとソフトウェアは現代の生活に欠かせません。プログラムを書く仕事は人気があります。
犬や猫は動物園でも見られます。鳥も動物園の人気者です。
ラーメンやパンは毎日の食事に使われます。寿司はお祝いの食べ物です。
コンピュータやソフトウェアは仕事で使われます。プログラムは便利な道具です。`

// contentPOS keeps semantically meaningful tags and drops function words
// (particles, auxiliary verbs, symbols) that add syntactic noise but no
// signal for meaning-based clustering.
var contentPOS = map[string]bool{
	hmm.POSNoun:    true,
	hmm.POSVerb:    true,
	hmm.POSAdj:     true,
	hmm.POSAdverb:  true,
	hmm.POSForeign: true,
	hmm.POSNumber:  true,
	hmm.POSUnknown: true,
}

func main() {
	dictFile, err := os.CreateTemp("", "morpho-embed-*.json")
	if err != nil {
		panic(err)
	}
	dictPath := dictFile.Name()
	dictFile.Close()
	defer os.Remove(dictPath)

	analyzer, err := morpho.New(dictPath)
	if err != nil {
		panic(err)
	}
	if err := analyzer.Train(corpus); err != nil {
		panic(err)
	}

	var sentences [][]string
	for _, line := range splitSentences(corpus) {
		morphemes, err := analyzer.Analyze(line)
		if err != nil {
			panic(err)
		}
		var words []string
		for _, m := range morphemes {
			if contentPOS[m.POS] {
				words = append(words, m.Surface)
			}
		}
		if len(words) > 0 {
			sentences = append(sentences, words)
		}
	}

	model := embedding.Train(sentences, embedding.DefaultConfig())

	fmt.Printf("学習語彙数: %d語 (次元数: %d)\n\n", len(model.Vectors), model.Dim)

	for _, seed := range []string{"犬", "ラーメン", "コンピュータ"} {
		neighbors := model.Nearest(seed, 5)
		if neighbors == nil {
			fmt.Printf("「%s」は語彙にありませんでした（分割結果を確認してください）\n\n", seed)
			continue
		}
		fmt.Printf("「%s」に意味が近い単語:\n", seed)
		for _, nb := range neighbors {
			fmt.Printf("  %-10s %.3f\n", nb.Word, nb.Score)
		}
		fmt.Println()
	}
}

// splitSentences breaks text into sentences at "。" and newlines so each
// sentence becomes one training context window for the embedding model.
func splitSentences(text string) []string {
	var out []string
	var buf []rune
	for _, r := range text {
		buf = append(buf, r)
		if r == '。' || r == '\n' {
			if s := string(buf); len([]rune(s)) > 1 {
				out = append(out, s)
			}
			buf = nil
		}
	}
	if len(buf) > 0 {
		out = append(out, string(buf))
	}
	return out
}
