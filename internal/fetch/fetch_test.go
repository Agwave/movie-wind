package fetch

import "testing"

func TestParseSumBox(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"12.97亿", 129700},
		{"8974.0万", 8974},
		{"3526", 0.3526},
		{"", 0},
	}
	for _, c := range cases {
		got, err := ParseSumBox(c.in)
		if c.in == "" {
			if err == nil {
				t.Fatal("empty should error")
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if got < c.want-0.001 || got > c.want+0.001 {
			t.Fatalf("%s: got %v want %v", c.in, got, c.want)
		}
	}
}

func TestParseDashboard(t *testing.T) {
	raw := []byte(`{"movieList":{"list":[{"avgSeatView":"2.1%","avgShowView":"3.1","boxRate":"31.5%","movieInfo":{"movieId":1462628,"movieName":"欢迎来龙餐馆","releaseInfo":"上映11天"},"showCount":143734,"showCountRate":"36.5%","sumBoxDesc":"12.97亿","sumSplitBoxDesc":"11.55亿"}]}}`)
	films, err := ParseDashboard(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(films) != 1 {
		t.Fatalf("len=%d", len(films))
	}
	f := films[0]
	if f.MovieID != 1462628 || f.Name != "欢迎来龙餐馆" {
		t.Fatalf("%+v", f)
	}
	if f.SumBoxWan < 129699 || f.SumBoxWan > 129701 {
		t.Fatalf("sum=%v", f.SumBoxWan)
	}
	if f.BoxRate != "31.5%" || f.ShowCount != 143734 {
		t.Fatalf("%+v", f)
	}
}
