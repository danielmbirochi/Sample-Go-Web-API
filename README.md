# Go Sample Service

Sample Go service for providing a base web app structure. It contemplates a clear startup/shutdown strategy,
database support, basic JWT authetication & authorization mechanism, proper structured log support using Uber 
zap pkg, and provides observability through OpenTelemetry tools using Zipkin. 
---
---

### **Human readable logs**

For proper formatting the structured logs in a human readable way, use the logfmt program under app/tooling folder.


### **Admin CLI Tool**

For getting the available admin CLI commands:
```
make admin-help
```

### **Service Configuration**

For getting the configset for running the service: 
```
make help
```
---