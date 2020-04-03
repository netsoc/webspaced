from flask_wtf import FlaskForm
from flask_wtf.file import FileField, FileAllowed
from wtforms import StringField, PasswordField, SubmitField, BooleanField
from wtforms.validators import DataRequired, Length, Email, EqualTo, ValidationError
from flask_login import current_user

class LoginForm(FlaskForm):
	email = StringField('Email', 
		validators=[DataRequired(), Length(min=2, max=20), Email()])
	password = PasswordField('Password', 
		validators = [DataRequired()])
	submit = SubmitField('Login')