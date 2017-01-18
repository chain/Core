import React from 'react'
import styles from './Success.scss'
import { Link } from 'react-router'

class Success extends React.Component {

  render() {
    const userInput = this.props.userInput
    const nextButton = <div className={styles.next}>
      <Link to={this.props.route}>
        <button key='showNext' className='btn btn-primary' onClick={this.props.handleNext}>
          {this.props.button}
        </button>
      </Link>
    </div>

    return (
      <div>
        <div className={styles.container}>
          <div className={styles.header}>
            {this.props.title}
            {this.props.dismiss &&
              <div className={styles.skip}>
                <a onClick={this.props.handleDismiss}>{this.props.dismiss}</a>
              </div>
            }
          </div>
          <div className={styles.content}>
            <span className='glyphicon glyphicon-ok-sign'></span>
            <div className={styles.text}>
              {this.props.content.map(function (x, i){
                var str = x.replace('STRING', userInput)
                return <li key={i}>{str}</li>
              })}
            </div>

            {nextButton && nextButton}
          </div>
        </div>
    </div>
    )
  }
}

export default Success
