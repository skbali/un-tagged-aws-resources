provider "aws" {
  region  = "us-east-1"
  profile = "default"
}

data "archive_file" "un_tagged_zip" {
  type        = "zip"
  output_path = "./code/code.zip"

  source_file = "./code/build/bootstrap"
}

resource "aws_lambda_function" "un_tagged" {
  function_name = "un-tagged-resources-go"
  handler       = "bootstrap"
  role          = aws_iam_role.un_tagged_lambda_role.arn

  runtime          = "provided.al2"
  timeout          = 300
  memory_size      = 128
  architectures   = ["arm64"]
  filename         = data.archive_file.un_tagged_zip.output_path
  source_code_hash = data.archive_file.un_tagged_zip.output_base64sha256

  environment {
    variables = {
      REGION    = var.region
      TOPIC_ARN = data.aws_sns_topic.site_monitor_sns_topic.arn
    }
  }
  tags = var.tags
}

resource "aws_cloudwatch_event_rule" "every_12_hours" {
  name                = "un-tagged-resources-go-12-hours"
  schedule_expression = "rate(12 hours)"
  tags                = var.tags
}

resource "aws_cloudwatch_event_target" "un_tagged_lambda_target" {
  arn  = aws_lambda_function.un_tagged.arn
  rule = aws_cloudwatch_event_rule.every_12_hours.name
}

resource "aws_lambda_permission" "un_tagged_lamabda_perms" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.un_tagged.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.every_12_hours.arn
}

data "aws_sns_topic" "site_monitor_sns_topic" {
  name = "site-monitor"
}