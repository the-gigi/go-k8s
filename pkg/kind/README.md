# kind

The `kind` package provides a Go interface for creating and deleting [kind]() clusters.

When creating multiple kind clusters make sure your Docker VM has enough CPU and memory.

For example, on the Mac using [colima](https://github.com/abiosoft/colima) for 3-4 kind clusters
It is recommended to use 4-6 CPUs and 8-12 GiB of memory. Here is how to start colima:

```
colima start --cpu 6 --memory 12
```

You can find node images here:
find the images with hash https://github.com/kubernetes-sigs/kind/releases

# Interesting project

https://github.com/Trendyol/kink