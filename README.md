# Seva-run 

Seva-run is a simple go based cpp runner that is intended for programming judging process.

Currently supports 
1. Compiling CPP
2. Comparing stdin and stdout
3. Time Limit
4. Memory Limit

---
## Steps to Run

1. Build
```sh
docker build -t seva-run .
```

2. Run
```sh
docker run --rm -p 8080:8080 \
              --cpus=1.0 \
              --memory=256m --memory-swap=256m \
              --pids-limit=128 \
              seva-run
```

And send POST request to `localhost:8080/judge`

---
## Sample Payload


### Accepted
```
{
  "source_cpp": "#include <bits/stdc++.h>\nusing namespace std;int main(){ios::sync_with_stdio(false);cin.tie(nullptr); long long n; if(!(cin>>n)) return 0; cout<<n*n<<\"\\n\"; }",
  "time_limit_ms": 1000,
  "memory_limit_mb": 128,
  "test_cases": [
    { "stdin": "7\n", "expected_stdout": "49" },
    { "stdin": "1000\n", "expected_stdout": "1000000" }
  ]
}
```

### Wrong Answer
```
{
  "source_cpp": "#include <bits/stdc++.h>\nusing namespace std;int main(){long long n; if(!(cin>>n)) return 0; cout << (n*n+1) << \"\\n\"; }",
  "time_limit_ms": 1000,
  "memory_limit_mb": 128,
  "test_cases": [
    { "stdin": "7\n", "expected_stdout": "49" }
  ]
}
```

### Time Limit Exception (TLE)
```
{
  "source_cpp": "#include <bits/stdc++.h>\nusing namespace std;int main(){volatile unsigned long long x=0; while(true){ x+=1; } }",
  "time_limit_ms": 100,
  "memory_limit_mb": 128,
  "test_cases": [
    { "stdin": "0\n", "expected_stdout": "" }
  ]
}
```


---

## Limitations
Memory limit is not fully stable, as i do not know how to it xD, but currently it will limit the memory such that the program is going to throw std::badalloc.

