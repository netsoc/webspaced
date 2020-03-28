import setuptools

with open('requirements.txt') as req_file:
    requirements = req_file.read()

setuptools.setup(
    name='webspace-ng',
    version='0.1.0',
    author="Jack O'Sullivan",
    author_email='osullj19@tcd.ie',
    description='Next generation webspace management',
    packages=setuptools.find_packages(),
    install_requires=requirements,
    entry_points={
        'console_scripts': [
            'webspace=webspace_cli:main',
        ]
    }
)
