data "aws_iam_policy_document" "un_tagged_lambda_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    effect  = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "un_tagged_lambda_role" {
  name               = "un-tagged-lambda"
  tags               = var.tags
  assume_role_policy = data.aws_iam_policy_document.un_tagged_lambda_assume.json
}

data "aws_iam_policy_document" "un_tagged_lambda_perms" {
  statement {
    sid    = "VisualEditor0"
    effect = "Allow"

    actions = [
      "logs:PutRetentionPolicy",
      "ec2:DescribeSnapshots",
      "ec2:DescribeVolumes",
      "ec2:DescribeInstances",
      "lambda:ListFunctions",
      "lambda:ListTags",
      "sns:Publish"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"

    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "un_tagged_lambda" {
  name   = "un-tagged-lambda"
  tags   = var.tags
  policy = data.aws_iam_policy_document.un_tagged_lambda_perms.json
}


resource "aws_iam_role_policy_attachment" "attach_ec2_policy" {
  policy_arn = aws_iam_policy.un_tagged_lambda.arn
  role       = aws_iam_role.un_tagged_lambda_role.name
}