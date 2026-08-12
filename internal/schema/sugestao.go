package schema

// limiteDeSugestao é a distância de Levenshtein máxima para que uma chave
// válida seja oferecida como sugestão de uma chave desconhecida.
const limiteDeSugestao = 3

// sugestao devolve a chave válida mais próxima de chave, ou "" quando nenhuma
// candidata está dentro do limite. Em empate vence a primeira na ordem
// declarada das chaves válidas.
func sugestao(chave string, validas []string) string {
	melhor := ""
	menor := limiteDeSugestao + 1
	for _, v := range validas {
		d := distancia(chave, v)
		if d < menor {
			melhor, menor = v, d
		}
	}
	return melhor
}

// distancia devolve a distância de Levenshtein entre a e b, contada em runas.
func distancia(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	anterior := make([]int, len(rb)+1)
	atual := make([]int, len(rb)+1)
	for j := range anterior {
		anterior[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		atual[0] = i
		for j := 1; j <= len(rb); j++ {
			custo := 1
			if ra[i-1] == rb[j-1] {
				custo = 0
			}
			atual[j] = min(anterior[j]+1, atual[j-1]+1, anterior[j-1]+custo)
		}
		anterior, atual = atual, anterior
	}
	return anterior[len(rb)]
}
