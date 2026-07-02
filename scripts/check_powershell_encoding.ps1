[CmdletBinding()]
param(
  [switch]$Fix
)

$ErrorActionPreference = 'Stop'

function Get-CodePage {
  if (-not (Get-Command chcp.com -ErrorAction SilentlyContinue)) {
    return $null
  }

  $line = & chcp.com
  if ($line -match '(\d+)') {
    return [int]$Matches[1]
  }
  return $null
}

function Set-Utf8Encoding {
  $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [Console]::InputEncoding = $utf8NoBom
  [Console]::OutputEncoding = $utf8NoBom
  $global:OutputEncoding = $utf8NoBom
  $global:PSDefaultParameterValues['Get-Content:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Set-Content:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Add-Content:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Out-File:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Select-String:Encoding'] = 'UTF8'

  if (Get-Command chcp.com -ErrorAction SilentlyContinue) {
    & chcp.com 65001 > $null
  }

  if (Get-Command git -ErrorAction SilentlyContinue) {
    Set-GitConfigIfNeeded 'core.quotepath' 'false' 'bool'
    Set-GitConfigIfNeeded 'i18n.logOutputEncoding' 'utf-8' 'encoding'
    Set-GitConfigIfNeeded 'i18n.commitEncoding' 'utf-8' 'encoding'
  }
}

function Set-GitConfigIfNeeded {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$Value,
    [ValidateSet('string', 'bool', 'encoding')][string]$ValueKind = 'string'
  )

  $current = Get-GitGlobalConfigValue $Name $ValueKind
  $expected = Convert-GitConfigExpectedValue $Value $ValueKind
  if ($current.NormalizedValue -ne $expected) {
    git config --global $Name $Value
    if ($LASTEXITCODE -ne 0) {
      throw "Failed to write global Git config '$Name'."
    }
  }
}

function Convert-GitConfigExpectedValue {
  param(
    [Parameter(Mandatory = $true)][string]$Value,
    [ValidateSet('string', 'bool', 'encoding')][string]$ValueKind = 'string'
  )

  if ($ValueKind -eq 'encoding') {
    return (($Value.ToLowerInvariant()) -replace '[-_]', '')
  }

  return $Value
}

function Get-GitGlobalConfigValue {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [ValidateSet('string', 'bool', 'encoding')][string]$ValueKind = 'string'
  )

  $value = git config --global --get $Name 2>$null
  if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrEmpty($value)) {
    $value = '<unset>'
  }

  $normalizedValue = [string]$value
  if ($ValueKind -eq 'bool' -and $value -ne '<unset>') {
    $boolValue = git config --global --type=bool --get $Name 2>$null
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrEmpty($boolValue)) {
      $normalizedValue = [string]$boolValue
    }
  } elseif ($ValueKind -eq 'encoding' -and $value -ne '<unset>') {
    $normalizedValue = Convert-GitConfigExpectedValue ([string]$value) 'encoding'
  }

  return [pscustomobject]@{
    DisplayValue = [string]$value
    NormalizedValue = $normalizedValue
  }
}

function Get-GitConfigCheckValue {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [ValidateSet('string', 'bool', 'encoding')][string]$ValueKind = 'string',
    [string]$DefaultValue = ''
  )

  if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    return [pscustomobject]@{
      DisplayValue = '<git not found>'
      NormalizedValue = '<git not found>'
      Source = '<git not found>'
    }
  }

  $value = git config --get $Name 2>$null
  if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrEmpty($value)) {
    $value = '<unset>'
  }

  $displayValue = [string]$value
  $normalizedValue = [string]$value

  if ($ValueKind -eq 'bool' -and $value -ne '<unset>') {
    $boolValue = git config --type=bool --get $Name 2>$null
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrEmpty($boolValue)) {
      $normalizedValue = [string]$boolValue
    }
  } elseif ($ValueKind -eq 'encoding') {
    if ($value -eq '<unset>' -and -not [string]::IsNullOrEmpty($DefaultValue)) {
      $displayValue = '<unset> (Git default utf-8)'
      $normalizedValue = Convert-GitConfigExpectedValue $DefaultValue 'encoding'
    } else {
      $normalizedValue = Convert-GitConfigExpectedValue ([string]$value) 'encoding'
    }
  }

  $origins = @(git config --show-origin --get-all $Name 2>$null)
  if ($LASTEXITCODE -ne 0 -or $origins.Count -eq 0) {
    $source = '<unset>'
  } else {
    $source = $origins -join '; '
  }

  return [pscustomobject]@{
    DisplayValue = $displayValue
    NormalizedValue = $normalizedValue
    Source = $source
  }
}

if ($Fix) {
  Set-Utf8Encoding
}

$codePage = Get-CodePage
$gitQuotePath = Get-GitConfigCheckValue 'core.quotepath' 'bool'
$gitLogEncoding = Get-GitConfigCheckValue 'i18n.logOutputEncoding' 'encoding' 'utf-8'
$gitCommitEncoding = Get-GitConfigCheckValue 'i18n.commitEncoding' 'encoding' 'utf-8'

function Test-DefaultEncoding {
  param([string]$Key)
  if ($PSDefaultParameterValues.ContainsKey($Key)) {
    return $PSDefaultParameterValues[$Key] -eq 'UTF8'
  }

  return $PSVersionTable.PSVersion.Major -ge 6
}

function Get-DefaultEncodingActual {
  param([string]$Key)
  if ($PSDefaultParameterValues.ContainsKey($Key)) {
    return [string]$PSDefaultParameterValues[$Key]
  }

  if ($PSVersionTable.PSVersion.Major -ge 6) {
    return '<PowerShell 6+ default UTF-8>'
  }

  return '<unset>'
}

$checks = @(
  [pscustomobject]@{
    Name = 'Active code page'
    Expected = '65001'
    Actual = if ($null -eq $codePage) { '<chcp.com not found>' } else { [string]$codePage }
    Pass = $codePage -eq 65001
  },
  [pscustomobject]@{
    Name = 'Console input encoding'
    Expected = 'Unicode (UTF-8)'
    Actual = [Console]::InputEncoding.EncodingName
    Pass = [Console]::InputEncoding.WebName -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'Console output encoding'
    Expected = 'Unicode (UTF-8)'
    Actual = [Console]::OutputEncoding.EncodingName
    Pass = [Console]::OutputEncoding.WebName -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'PowerShell pipe output encoding'
    Expected = 'Unicode (UTF-8)'
    Actual = $OutputEncoding.EncodingName
    Pass = $OutputEncoding.WebName -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'Git quoted paths'
    Expected = 'false'
    Actual = $gitQuotePath.DisplayValue
    Source = $gitQuotePath.Source
    Pass = $gitQuotePath.NormalizedValue -eq 'false'
  },
  [pscustomobject]@{
    Name = 'Git log output encoding'
    Expected = 'utf-8'
    Actual = $gitLogEncoding.DisplayValue
    Source = $gitLogEncoding.Source
    Pass = $gitLogEncoding.NormalizedValue -eq 'utf8'
  },
  [pscustomobject]@{
    Name = 'Git commit encoding'
    Expected = 'utf-8'
    Actual = $gitCommitEncoding.DisplayValue
    Source = $gitCommitEncoding.Source
    Pass = $gitCommitEncoding.NormalizedValue -eq 'utf8'
  },
  [pscustomobject]@{
    Name = 'Get-Content default encoding'
    Expected = 'UTF8'
    Actual = Get-DefaultEncodingActual 'Get-Content:Encoding'
    Pass = Test-DefaultEncoding 'Get-Content:Encoding'
  },
  [pscustomobject]@{
    Name = 'Select-String default encoding'
    Expected = 'UTF8'
    Actual = Get-DefaultEncodingActual 'Select-String:Encoding'
    Pass = Test-DefaultEncoding 'Select-String:Encoding'
  },
  [pscustomobject]@{
    Name = 'Out-File default encoding'
    Expected = 'UTF8'
    Actual = Get-DefaultEncodingActual 'Out-File:Encoding'
    Pass = Test-DefaultEncoding 'Out-File:Encoding'
  }
)

$checks | Format-Table Name, Expected, Actual, Source, Pass -AutoSize

if ($checks.Pass -contains $false) {
  $failedGitChecks = @($checks | Where-Object {
      -not $_.Pass -and $_.Name -like 'Git *'
    })
  if ($failedGitChecks.Count -gt 0) {
    Write-Host ''
    Write-Host 'Git config sources for failed checks:'
    foreach ($check in $failedGitChecks) {
      Write-Host ('- {0}: {1}' -f $check.Name, $check.Source)
    }
  }

  if ($Fix) {
    Write-Error 'PowerShell UTF-8 encoding checks failed after -Fix. For Git checks, review the Source column for local/global/system overrides and update the overriding config.'
  } else {
    Write-Error 'PowerShell UTF-8 encoding checks failed. Run in the affected PowerShell session: .\scripts\check_powershell_encoding.ps1 -Fix. For Git checks, review the Source column for local/global/system overrides.'
  }
}
