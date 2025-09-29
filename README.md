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
Output:
```
{
  "compile_ok": true,
  "results": [
    {
      "status": "AC",
      "stdout": "49\n",
      "time_ms": 4,
      "max_rss_kb": 9688
    },
    {
      "status": "AC",
      "stdout": "1000000\n",
      "max_rss_kb": 9720
    }
  ]
}
```

### Wrong Answer
Input:
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
Output:
```
{
  "compile_ok": true,
  "results": [
    {
      "status": "WA",
      "stdout": "50\n",
      "reason": "expected \"49\", got \"50\"",
      "time_ms": 1,
      "max_rss_kb": 9672
    }
  ]
}
```



### Time Limit Exception (TLE)
Infinite loop
Input:
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
Output:
```
{
  "compile_ok": true,
  "results": [
    {
      "status": "TLE",
      "reason": "exceeded 100ms",
      "time_ms": 102,
      "max_rss_kb": 9620
    }
  ]
}
```



### Runtime Exception 
This will result in segfault
Input:
```
{
  "source_cpp": "#include <bits/stdc++.h>\nusing namespace std;int main(){int* p=nullptr; *p=42; return 0;}",
  "time_limit_ms": 1000,
  "memory_limit_mb": 128,
  "test_cases": [
    { "stdin": "0\n", "expected_stdout": "" }
  ]
}
```
Output:
```
{
  "compile_ok": true,
  "results": [
    {
      "status": "RTE",
      "reason": "terminated by segmentation fault",
      "exit_code": -1,
      "signal": "segmentation fault",
      "max_rss_kb": 9284
    }
  ]
}
```

---

## Limitations
Memory limit is not fully stable, as i do not know how to it xD, but currently it will limit the memory such that the program is going to throw std::badalloc.

---
## Memory RSS Test

With this payload tests memory buffer allocation for 32 KB, 64 KB, and 96KB
```
{
  "source_cpp": "#include <bits/stdc++.h>\nusing namespace std;int main(){ios::sync_with_stdio(false);cin.tie(nullptr); size_t mb; if(!(cin>>mb)) return 0; size_t bytes = mb*1024ull*1024ull; vector<char> buf(bytes); for(size_t i=0;i<bytes;i+=4096) buf[i]=1; this_thread::sleep_for(chrono::milliseconds(200)); cout<<\"OK\\n\"; }",
  "time_limit_ms": 2000,
  "memory_limit_mb": 128,
  "test_cases": [
    { "stdin": "32\n", "expected_stdout": "OK" },
    { "stdin": "64\n", "expected_stdout": "OK" },
    { "stdin": "96\n", "expected_stdout": "OK" }
  ]
}
```

This program outputs
```
{
  "compile_ok": true,
  "results": [
    {
      "status": "AC",
      "stdout": "OK\n",
      "time_ms": 212,
      "max_rss_kb": 35496
    },
    {
      "status": "AC",
      "stdout": "OK\n",
      "time_ms": 224,
      "max_rss_kb": 68320
    },
    {
      "status": "AC",
      "stdout": "OK\n",
      "time_ms": 224,
      "max_rss_kb": 101092
    }
  ]
}
```
