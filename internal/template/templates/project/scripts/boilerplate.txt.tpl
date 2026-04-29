{{- $author := .Project.Metadata.Author.Name | default "Anonymous" -}}
{{- $email := .Project.Metadata.Author.Email | default "noreply@example.com" -}}
Copyright {{ now | date "2006" }} {{ $author }} <{{ $email }}>. All rights reserved.
Use of this source code is governed by a MIT style license that can be
found in the LICENSE file.
The original repo for this file is {{ .Project.Metadata.Module }}
