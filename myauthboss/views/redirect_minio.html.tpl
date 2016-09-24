
{{$loggedin := .loggedin}}

{{if $loggedin}}

{{$minioToken := .minioToken}}
{{if $minioToken}}
<script type="text/javascript">
	localStorage.token = {{$minioToken}}
	window.location.replace("/")
</script>
{{end}}

{{else}}
<script type="text/javascript">
  localStorage.token = ""
	window.location.replace("/auth/login")
</script>
{{end}}
