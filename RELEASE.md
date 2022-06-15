## Release Note 🍻

### Fixes 🛠

- Delete the needless preStop command, there is a risk will that failed. When persistence is enabled, preStop failure can lead to inconsistent EMQX cluster data
